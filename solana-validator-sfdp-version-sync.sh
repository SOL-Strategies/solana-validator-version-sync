#!/bin/bash
set -euo pipefail

# program config
declare -A CONFIG=(
    ["activePubkey"]=""
    ["binaryPath"]=""
    ["buildScript"]="/home/solana/bin/solana-validator-source.sh"
    ["modsEnabled"]="false"
    ["client"]=""
    ["cluster"]=""
    ["cmd"]=""
    ["configFile"]="/home/solana/solana-validator-sfdp-version-sync/config.json"
    ["gossip_pubkey"]=""
    ["gossip_version"]=""
    ["installed_version"]=""
    ["logLevel"]="info"
    ["modName"]="default"
    ["modsIgnoreIncompatibleSemver"]="false"
    ["passivePubkey"]=""
    ["prometheusMetricsFile"]="/home/solana/solana-validator-prometheus-exporter/metrics/solana_validator_sfdp_version_sync.prom"
    ["publicIP"]="$(curl -s -4 ifconfig.me)"
    ["rebuildAlways"]="false"
    ["restartServiceAfterVersionSync"]="false"
    ["role"]=""
    ["rpcAddress"]="http://localhost:8899"
    ["serviceName"]=""
    ["sfdp_max_version"]=""
    ["sfdp_min_version"]=""
    ["syncIntervalSeconds"]="600" # every 10 minutes should be enough
)

# prometheus labels
declare -A PROMETHEUS_LABELS=(
    ["sync_status"]="unknown"
)

# convenience log wrapper
logger() {
    local level="$1"
    local prefix="$2"
    shift 2
    TZ=utc gum log --structured --time rfc3339 --level "$level" --prefix "$prefix" --min-level "${CONFIG["logLevel"]}" "$@"
}

# print usage
print_usage () {
    local exit_code="${1:-0}"
    cat <<EOF >&2
Keep the running version of the validator in sync with sfdp version requirements.

solana-validator-sfdp-version-sync.sh [command] [flags]

Commands:
    run  run the sync process (runs on a loop, updating when required based on --config)
         it will dump a prometheus metric on the sync status for our prometheus exporter into:
         ${CONFIG["prometheusMetricsFile"]}

Flags:
    --config        <path>   path to config file (default: ${CONFIG["configFile"]})
    -l, --log-level <level>  log level (default: info)
    -h,  --help
EOF
    exit "$exit_code"
}

# parse args
parse_args () {
    # if no args are provided, print usage
    [ $# -eq 0 ] && print_usage 1

    # parse args
    while [ $# -gt 0 ]; do
        case $1 in
            --config)
                CONFIG["configFile"]="$2"
                shift 2
                ;;
            --log-level|-l)
                CONFIG["logLevel"]="$2"
                shift 2
                ;;
            run)
                CONFIG["cmd"]="run"
                shift
                ;;
            --help|-h)
                print_usage 0
                ;;
            *)
                logger fatal parse_args "Unknown argument: $1"
                ;;
        esac
    done
}

require () {
    logger info require "checking requirements"

    # require root
    if [ "$(id -u)" != "0" ]; then
        logger fatal require "This script must be run as root because it may run systemctl commands and do other things that require root"
    fi

    local cmds=(jq solana solana-keygen agave-validator)
    for cmd in "${cmds[@]}"; do
        logger debug require "checking for command" cmd "${cmd}"
        if ! which "${cmd}" > /dev/null 2>&1; then
            logger fatal require "Command ${cmd} could not be found and is required for running solana-validator-sfdp-version-sync"
        fi
    done
}

configure () {
    logger info configure "🔧 configuring" config_file "${CONFIG["configFile"]}"

    write_prometheus_metric_with_status "configuring"

    if [ ! -f "${CONFIG["configFile"]}" ]; then
        write_fatal_prometheus_metric configure "config file ${CONFIG["configFile"]} does not exist"
    fi

    # load config json
    local config_json="$(jq -rc '.' "${CONFIG["configFile"]}")"

    # read flags from config.json
    for key in $(jq -r '. | keys[]' <<< "${config_json}"); do
        # if key is prometheusLabels, read all labels into PROMETHEUS_LABELS
        if [ "${key}" == "prometheusLabels" ]; then
            for label in $(jq -r '.prometheusLabels | keys[]' <<< "${config_json}"); do
                PROMETHEUS_LABELS["${label}"]="$(jq -r ".prometheusLabels[\"${label}\"]" <<< "${config_json}")"
            done
            continue
        fi

        # read root keys into CONFIG
        if [ "$(jq -r ".${key}" <<< "${config_json}")" != "null" ]; then
            local value="$(jq -r ".${key}" <<< "${config_json}")"

            case "${key}" in
                syncIntervalSeconds)
                    if ! [[ "${value}" =~ ^[0-9]+$ ]]; then
                        write_fatal_prometheus_metric configure "${key} must be a positive integer, got ${value}"
                    fi
                    ;;
                cluster)
                    if [ "${value}" != "testnet" ] && [ "${value}" != "mainnet-beta" ]; then
                        write_fatal_prometheus_metric configure "cluster must be one of: testnet, mainnet-beta"
                    fi
                    ;;
                serviceName)
                    # service must exist
                    if ! systemctl list-unit-files | grep -w "${value}" > /dev/null 2>&1; then
                        write_fatal_prometheus_metric configure "service name ${value} is not an existing service from systemctl"
                    fi
                    # service must be enabled
                    if ! systemctl is-enabled "${value}" | grep -q "enabled"; then
                        write_fatal_prometheus_metric configure "service ${value} is not enabled"
                    fi
                    # service must be active
                    if ! systemctl is-active "${value}" | grep -q "active"; then
                        write_fatal_prometheus_metric configure "service ${value} is not running"
                    fi
                    ;;
            esac

            # config value is valid
            CONFIG["${key}"]="${value}"
        fi
    done

    # get active pubkey
    logger debug configure "getting active pubkey" keyfile "${CONFIG["activeIdentityKeyfile"]}"
    CONFIG["activePubkey"]="$(solana-keygen pubkey "${CONFIG["activeIdentityKeyfile"]}")"

    # get passive pubkey
    logger debug configure "getting passive pubkey" keyfile "${CONFIG["passiveIdentityKeyfile"]}"
    CONFIG["passivePubkey"]="$(solana-keygen pubkey "${CONFIG["passiveIdentityKeyfile"]}")"

    # query sfdp api for validator - its active pubkey should be in SFDP
    logger debug configure "checking if validator is in SFDP"
    local sfdp_validator_api_url="https://api.solana.org/api/validators/${CONFIG["activePubkey"]}"
    local sfdp_validator_json="$(curl -s -X GET "${sfdp_validator_api_url}")"

    if [ "$(jq -r '.error' <<< "${sfdp_validator_json}")" != "null" ]; then
        logger warn configure "validator not found in SFDP api" \
            error "$(jq -r '.message' <<< "${sfdp_validator_json}")" \
            url "${sfdp_validator_api_url}" \
            pubkey "${CONFIG["activePubkey"]}"
    else
        logger debug configure "validator found in SFDP api" state "$(jq -r '.state' <<< "${sfdp_validator_json}")" pubkey "${CONFIG["activePubkey"]}"
    fi

    # set binary path and detect jito
    local installed_version_output=
    case "${CONFIG["client"]}" in
        agave)
            CONFIG["binaryPath"]="/usr/local/bin/agave-validator"
            installed_version_output="$("${CONFIG["binaryPath"]}" --version)"
            CONFIG["installed_version"]="$(echo "${installed_version_output}" | awk '{print $2}')"
            # if current version contains "JitoLabs", set client to jito
            if echo "${installed_version_output}" | grep -q -w "JitoLabs"; then
                logger debug configure "detected jito agave validator" binary_path "${CONFIG["binaryPath"]}" version "${installed_version_output}"
                CONFIG["client"]="jito"
            else
                logger debug configure "detected agave validator" binary_path "${CONFIG["binaryPath"]}" version "${installed_version_output}"
            fi
            ;;
        firedancer)
            CONFIG["binaryPath"]="/usr/local/bin/fdctl"
            installed_version_output="$("${CONFIG["binaryPath"]}" --version)"
            CONFIG["installed_version"]="$(echo "${installed_version_output}" | awk '{print $1}')"
            logger debug configure "detected firedancer validator" binary_path "${CONFIG["binaryPath"]}" version "${installed_version_output}"
            ;;
        *)
            write_fatal_prometheus_metric configure "client must be one of: agave, firedancer"
    esac

    # get current gossip entry - wait until validator appears in gossip if needed
    logger info configure "👀 looking for validator in gossip by public IP" public_ip "${CONFIG["publicIP"]}"
    local gossipResult="$(solana -u${CONFIG["cluster"]:0:1} gossip | grep -w "${CONFIG["publicIP"]}" || echo "")"
    while [ -z "$gossipResult" ]; do
        gossipResult="$(solana -u${CONFIG["cluster"]:0:1} gossip | grep -w "${CONFIG["publicIP"]}" || echo "")"
        logger warn configure "👀 looking for validator in gossip, retrying in 30 seconds" public_ip "${CONFIG["publicIP"]}"
        sleep 30
    done
    logger info configure "👉 validator found in gossip" public_ip "${CONFIG["publicIP"]}"

    # get pubkey and version from gossip
    CONFIG["gossip_pubkey"]="$(echo "$gossipResult" | awk '{print $3}')"
    # sometimes the version has a | character at the end so strip it.
    CONFIG["gossip_version"]="$(echo "$gossipResult" | awk '{print $13}' | cut -d "|" -f 1)"

    # dtermine role from gossip pubkey
    if [ "${CONFIG["gossip_pubkey"]}" == "${CONFIG["activePubkey"]}" ]; then
        CONFIG["role"]="active"
    elif [ "${CONFIG["gossip_pubkey"]}" == "${CONFIG["passivePubkey"]}" ]; then
        CONFIG["role"]="passive"
    else
        write_fatal_prometheus_metric configure "node is running with a pubkey that does not match active or passive pubkey" \
            gossip_pubkey "${CONFIG["gossip_pubkey"]}" \
            active_pubkey "${CONFIG["activePubkey"]}" \
            passive_pubkey "${CONFIG["passivePubkey"]}"
    fi

    # get requirements from sfdp api
    local sfdp_requirements_api_url="https://api.solana.org/api/epoch/required_versions?cluster=${CONFIG["cluster"]}"
    logger debug configure "getting requirements from sfdp api" url "${sfdp_requirements_api_url}"

    local requirements_json="$(curl -s -X GET "${sfdp_requirements_api_url}")"
    if [ "$(jq -r '.error' <<< "${requirements_json}")" != "null" ]; then
        write_fatal_prometheus_metric configure "error getting requirements from sfdp api" \
            error "$(jq -r '.message' <<< "${requirements_json}")" \
            url "${sfdp_requirements_api_url}"
    fi

    # lookup the min and max versions from sfdp api for the given client
    # if client is jito, lookup agave version bounds
    local sfdp_lookup_client="${CONFIG["client"]}"
    if [ "${CONFIG["client"]}" == "jito" ]; then
        sfdp_lookup_client="agave"
    fi
    logger debug configure "latest requirements from sfdp api" data "${requirements_json}"
    CONFIG["sfdp_min_version"]="$(jq -r ".data[-1][\"${sfdp_lookup_client}_min_version\"]" <<< "${requirements_json}")"
    CONFIG["sfdp_max_version"]="$(jq -r ".data[-1][\"${sfdp_lookup_client}_max_version\"]" <<< "${requirements_json}")"

    # set prometheus labels from gathered info
    PROMETHEUS_LABELS["client"]="${CONFIG["client"]}"
    PROMETHEUS_LABELS["role"]="${CONFIG["role"]}"
    PROMETHEUS_LABELS["pubkey"]="${CONFIG["gossip_pubkey"]}"
    PROMETHEUS_LABELS["running_version"]="${CONFIG["gossip_version"]}"
    PROMETHEUS_LABELS["installed_version"]="${CONFIG["installed_version"]}"
    PROMETHEUS_LABELS["sfdp_min_version"]="${CONFIG["sfdp_min_version"]}"
    PROMETHEUS_LABELS["sfdp_max_version"]="${CONFIG["sfdp_max_version"]}"

    # add prometheus labels from config if provided
    if [ "$(jq -r '.prometheusLabels' <<< "${config_json}")" != "null" ]; then
        for key in $(jq -r '.prometheusLabels | keys[]' <<< "${config_json}"); do
            PROMETHEUS_LABELS["${key}"]="$(jq -r ".prometheusLabels[\"${key}\"]" <<< "${config_json}")"
        done
    fi

    logger info configure "🔧 configured" \
        active_pubkey "${CONFIG["activePubkey"]}" \
        client "${CONFIG["client"]}" \
        cluster "${CONFIG["cluster"]}" \
        gossip_pubkey "${CONFIG["gossip_pubkey"]}" \
        gossip_version "${CONFIG["gossip_version"]}" \
        installed_version "${CONFIG["installed_version"]}" \
        mods_enabled "${CONFIG["modsEnabled"]}" \
        public_ip "${CONFIG["publicIP"]}" \
        role "${CONFIG["role"]}" \
        rpc_address "${CONFIG["rpcAddress"]}" \
        service "${CONFIG["serviceName"]}" \
        sfdp_min_version "${CONFIG["sfdp_min_version"]}" \
        sfdp_max_version "${CONFIG["sfdp_max_version"]}" \
        sync_interval_seconds "${CONFIG["syncIntervalSeconds"]}"
}

# runs a sync loop that continually loads config file, gets the node state and decides whether to upgrade, downgrade or do nothing
sync () {
    logger info sync "starting sync loop every ${CONFIG["syncIntervalSeconds"]} seconds"
    while true; do
        logger info "" "================================================"
        configure
        logger info sync "♻️ starting sync" \
            cluster "${CONFIG["cluster"]}" \
            role "${CONFIG["role"]}" \
            gossip_version "${CONFIG["gossip_version"]}"
        write_prometheus_metric_with_status "syncing"
        local sync_action="$(get_sync_action)"

        # if mainnet and role not passive - skip sync to be safe (we check this in sync_version too to be double sure)
        if [ "${CONFIG["cluster"]}" == "mainnet-beta" ] && [ "${CONFIG["role"]}" != "passive" ]; then
            logger warn sync "⚠️ node not passive, skipping sync and retrying in ${CONFIG["syncIntervalSeconds"]} seconds" \
                sync_action "${sync_action}" \
                cluster "${CONFIG["cluster"]}" \
                role "${CONFIG["role"]}" \
                gossip_version "${CONFIG["gossip_version"]}"
            write_prometheus_metric_with_status "synced"
            sleep "${CONFIG["syncIntervalSeconds"]}"
            continue
        fi

        # not on mainnet or role is passive - do the sync
        case "${sync_action}" in
            downgrade)
                logger warn sync "👎 downgrading to ${CONFIG["sfdp_max_version"]}" \
                    client "${CONFIG["client"]}" \
                    gossip_version "${CONFIG["gossip_version"]}" \
                    sfdp_max_version "${CONFIG["sfdp_max_version"]}"
                sync_version "downgrade" "${CONFIG["sfdp_max_version"]}"
                ;;
            upgrade)
                logger warn sync "👍 upgrading to ${CONFIG["sfdp_min_version"]}" \
                    client "${CONFIG["client"]}" \
                    gossip_version "${CONFIG["gossip_version"]}" \
                    sfdp_min_version "${CONFIG["sfdp_min_version"]}"
                sync_version "upgrade" "${CONFIG["sfdp_min_version"]}"
                ;;
            nochange)
                logger info sync "👌 running version ${CONFIG["gossip_version"]} is within SFDP version bounds" \
                    client "${CONFIG["client"]}" \
                    gossip_version "${CONFIG["gossip_version"]}" \
                    sfdp_min_version "${CONFIG["sfdp_min_version"]}" \
                    sfdp_max_version "${CONFIG["sfdp_max_version"]}"
                ;;
            *)
                logger fatal sync "⚠️ unknown sync action" action "${sync_action}"
        esac

        write_prometheus_metric_with_status "synced"
        logger info sync "✅ done - next sync in ${CONFIG["syncIntervalSeconds"]} seconds"
        sleep "${CONFIG["syncIntervalSeconds"]}"
    done
}

sync_version () {
    local action="${1}"
    local target_version="${2}"
    local build_success="false"

    write_prometheus_metric_with_status "syncing"

    # if mainnet and role not passive, warn and don't do anything
    if [ "${CONFIG["cluster"]}" == "mainnet-beta" ] && [ "${CONFIG["role"]}" != "passive" ]; then
        logger warn sync_version "mainnet version require passive role - skipping ${action} sync" \
            role "${CONFIG["role"]}" \
            target_version "${target_version}" \
            gossip_version "${CONFIG["gossip_version"]}"
        return
    fi

    logger info sync_version "version ${action} to ${target_version}"

    # if installed version is already the target version and no mods - skip build
    if [ "${CONFIG["installed_version"]}" == "${target_version}" ] && [ "${CONFIG["rebuildAlways"]}" == "false" ]; then
        logger warn sync_version "installed version already at ${target_version} - skipping build"
        build_success="true"
    else
        # tell us we're building
        logger info sync_version "building version ${target_version}" installed_version "${CONFIG["installed_version"]}" mods_enabled "${CONFIG["modsEnabled"]}"

        if [ "${CONFIG["modsEnabled"]}" == "true" ]; then
            logger info sync_version "mods are enabled" mod_name "${CONFIG["modName"]}"
        fi

        # base build command
        local build_command_args=("build" "--client" "${CONFIG["client"]}" "--version" "${target_version}")

        # if downgrading, add --allow-downgrade flag
        if [ "${action}" == "downgrade" ]; then
            build_command_args+=("--allow-downgrade")
        fi

        # if mods are enabled, add the mods cluster and name to the build command
        if [ "${CONFIG["modsEnabled"]}" == "true" ]; then
            # strip -beta suffix from cluster name for mods
            local mods_cluster="${CONFIG["cluster"]%-beta}"
            build_command_args+=("--mods-enabled" "--mods-cluster" "${mods_cluster}" "--mod-name" "${CONFIG["modName"]}")
            # if the referenced --mod-name is not compatible with the target version, add --mods-ignore-incompatible-semver flag
            if [ "${CONFIG["modsIgnoreIncompatibleSemver"]}" == "true" ]; then
                build_command_args+=("--mods-ignore-incompatible-semver")
            fi
        fi

        # run the build command
        logger info sync_version "building version ${target_version}" command "${CONFIG["buildScript"]}" args "${build_command_args[*]}"
        "${CONFIG["buildScript"]}" "${build_command_args[@]}" || write_fatal_prometheus_metric sync_version "failed to build version ${target_version}"
        build_success="true"
        logger info sync_version "build - great success!"
    fi

    # double sure to catch build failure
    if [ "${build_success}" == "false" ]; then
        write_fatal_prometheus_metric sync_version "failed to build version ${target_version}"
    fi

    # install command args
    local install_command_args=("install" "--client" "${CONFIG["client"]}" "--version" "${target_version}")
    logger info sync_version "installing version ${target_version}" command "${CONFIG["buildScript"]}" args "${install_command_args[*]}"
    "${CONFIG["buildScript"]}" "${install_command_args[@]}" || write_fatal_prometheus_metric sync_version "failed to install version ${target_version}"

    # restart service if enabled
    if [ "${CONFIG["restartServiceAfterVersionSync"]}" == "true" ]; then
        logger info sync_version "restarting service ${CONFIG["serviceName"]}"

        systemctl restart "${CONFIG["serviceName"]}" || write_fatal_prometheus_metric sync_version "failed to restart service ${CONFIG["serviceName"]}"

        logger info sync_version "restarted service ${CONFIG["serviceName"]}, follow logs with: journalctl -u ${CONFIG["serviceName"]} -f -o cat"

        # monitor the service until it receives an "ok" response from curl /health
        logger info sync_version "waiting for healthy response from validator, retrying silently every 10 seconds..." service "${CONFIG["serviceName"]}" rpc_address "${CONFIG["rpcAddress"]}"
        while [ "$(curl -s --max-time 5 "${CONFIG["rpcAddress"]}/health")" != "ok" ]; do
            logger debug sync_version "service ${CONFIG["serviceName"]} not ready, retrying in 10 seconds"
            sleep 10
        done

        logger info sync_version "service ${CONFIG["serviceName"]} is healthy"
    else
        logger info sync_version "restarting service ${CONFIG["serviceName"]} is disabled - skipping restart"
    fi

    logger info sync_version "done - upgraded to ${target_version}"
}

# compares gossip version to sfdp min and max versions and returns "upgrade", "downgrade" or "nochange"
get_sync_action () {
    if [ "${CONFIG["sfdp_max_version"]}" != "null" ]; then
        logger debug get_sync_action "comparing gossip version to sfdp max version" gossip_version "${CONFIG["gossip_version"]}" sfdp_max_version "${CONFIG["sfdp_max_version"]}"
        if dpkg --compare-versions "${CONFIG["gossip_version"]}" gt "${CONFIG["sfdp_max_version"]}"; then
            echo "downgrade"
            return
        fi
    fi

    if [ "${CONFIG["sfdp_min_version"]}" != "null" ]; then
        logger debug get_sync_action "comparing gossip version to sfdp min version" gossip_version "${CONFIG["gossip_version"]}" sfdp_min_version "${CONFIG["sfdp_min_version"]}"
        if dpkg --compare-versions "${CONFIG["gossip_version"]}" lt "${CONFIG["sfdp_min_version"]}"; then
            echo "upgrade"
            return
        fi
    fi

    echo "nochange"
}

# writes a prometheus metric solana_validator_sfdp_version_sync{labelx="labelxvalue",labely="labelyvalue"} 1
write_prometheus_metric_with_status () {
    # if the metrics file dir doesn't exist, warn and move on
    if [ ! -d "$(dirname "${CONFIG["prometheusMetricsFile"]}")" ]; then
        logger error prometheus "metrics file directory does not exist, skipping metric write" file "${CONFIG["prometheusMetricsFile"]}"
        return
    fi

    PROMETHEUS_LABELS["sync_status"]="${1:-unknown}"
    PROMETHEUS_LABELS["public_ip"]="${CONFIG["publicIP"]}"

    logger debug prometheus "writing metric solana_validator_sfdp_version_sync"
    # build labels string from associative array
    local labels_string=""
    local first=true
    for key in "${!PROMETHEUS_LABELS[@]}"; do
        local value="${PROMETHEUS_LABELS[$key]}"
        if [ "$first" = true ]; then
            labels_string="${key}=\"${value}\""
            first=false
        else
            labels_string="${labels_string},${key}=\"${value}\""
        fi
    done
    local metric="solana_validator_sfdp_version_sync{${labels_string}} 1"
    local metrics_file_tmp="${CONFIG["prometheusMetricsFile"]}.tmp"
    # write the metric
    logger debug prometheus "exporting metric" metric "${metric}" file "${CONFIG["prometheusMetricsFile"]}"
    echo "${metric}" > "${metrics_file_tmp}"
    # move the tmp file to the metrics file
    mv "${metrics_file_tmp}" "${CONFIG["prometheusMetricsFile"]}"
}

# same as write_prometheus_metric_with_status, but with a status of fatal
write_fatal_prometheus_metric () {
    local prefix="${1}"
    local message="${2}"
    shift 2
    write_prometheus_metric_with_status "fatal"
    logger fatal "${prefix}" "${message}" $@
}

main () {
    parse_args "$@"
    case "${CONFIG["cmd"]}" in
        run)
            logger info "" "================================================"
            logger info "" "🚀 starting solana-validator-sfdp-version-sync"
            logger info "" "================================================"
            require
            sync
            ;;
        *)
            print_usage 1
            ;;
    esac
}

main "$@"
