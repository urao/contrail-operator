#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

usage ()
{
    echo "usage: $0 [--dir <output-dir>][--config <config file>][--operator-dir <contrail-operator project directory>]"
}

get_parameters() {
	while [ ! $# -eq 0 ]
	do
		case "$1" in
            --help)
                usage
                exit 0
                ;;
			--dir)
				DIRECTORY=$2
				;;
			--operator-dir)
				OPERATOR_DIR=$2
				;;
			--config)
				CONFIG=$2
                ;;
		esac
		shift
	done

	if [ -z $OPERATOR_DIR ]
	then
        OPERATOR_DIR="${SCRIPT_DIR}/../../"
	fi

    if [ -z $CONFIG ]
    then
       CONFIG="${SCRIPT_DIR}/config"
    fi
}

copy_manifests() {
	mkdir -p "$DIRECTORY/openshift"
	cp -v ${SCRIPT_DIR}/openshift/* "${DIRECTORY}/openshift"
	mkdir -p "$DIRECTORY/manifests"
	cp -v ${SCRIPT_DIR}/manifests/* "${DIRECTORY}/manifests"
	echo "[INFO] Manifests have been copied to ${DIRECTORY}"
}

copy_and_rename_crds() {
	for f in ${OPERATOR_DIR}/deploy/crds/*_crd.yaml;
	do
		f_filename=$(basename $f)
		cp -v ${f} "${DIRECTORY}/manifests/00-contrail-07-${f_filename}"
	done
	echo '[INFO] Manifests CRDs have been properly renamed'
}

read_config() {
    if [ ! -f "$CONFIG" ]; then
        echo "$CONFIG file does not exist. Please provide config file."
        exit 1
    fi

    CONTRAIL_VERSION=$(grep "CONTRAIL_VERSION" $CONFIG)
    if [ $? -ne 0 ]; then
        echo "Couldn't find CONTRAIL_VERSION parameter. Exiting..."
        exit 1
    fi
    DOCKER_CONFIG=$(grep "DOCKER_CONFIG" $CONFIG)
    if [ $? -ne 0 ]; then
        echo "Couldn't find DOCKER_CONFIG parameter. Exiting..."
        exit 1
    fi
    CONTRAIL_REGISTRY=$(grep "CONTRAIL_REGISTRY" $CONFIG)
    if [ $? -ne 0 ]; then
        echo "Couldn't find CONTRAIL_REGISTRY parameter. Exiting..."
        exit 1
    fi
    CONTRAIL_VERSION="${CONTRAIL_VERSION##CONTRAIL_VERSION=}"
    DOCKER_CONFIG="${DOCKER_CONFIG##DOCKER_CONFIG=}"
    CONTRAIL_REGISTRY="${CONTRAIL_REGISTRY##CONTRAIL_REGISTRY=}"
    echo '[INFO] Config properly consumed'
}

apply_config() {
    sed -i.bak 's|<CONTRAIL_VERSION>|'$CONTRAIL_VERSION'|g' ${DIRECTORY}/manifests/00-contrail-08-operator.yaml && rm ${DIRECTORY}/manifests/00-contrail-08-operator.yaml.bak
    sed -i.bak 's|<CONTRAIL_VERSION>|'$CONTRAIL_VERSION'|g' ${DIRECTORY}/manifests/00-contrail-09-manager.yaml && rm ${DIRECTORY}/manifests/00-contrail-09-manager.yaml.bak
    sed -i.bak 's|<CONTRAIL_REGISTRY>|'$CONTRAIL_REGISTRY'|g' ${DIRECTORY}/manifests/00-contrail-08-operator.yaml && rm ${DIRECTORY}/manifests/00-contrail-08-operator.yaml.bak
    sed -i.bak 's|<CONTRAIL_REGISTRY>|'$CONTRAIL_REGISTRY'|g' ${DIRECTORY}/manifests/00-contrail-09-manager.yaml && rm ${DIRECTORY}/manifests/00-contrail-09-manager.yaml.bak
    sed -i.bak 's|<DOCKER_CONFIG>|'$DOCKER_CONFIG'|g' ${DIRECTORY}/manifests/00-contrail-02-registry-secret.yaml && rm ${DIRECTORY}/manifests/00-contrail-02-registry-secret.yaml.bak
    echo '[INFO] Set proper parameters from config in manifests'
}

DIRECTORY=$(pwd)
get_parameters "$@"
echo '[INFO] Starting setup script'
read_config
copy_manifests
copy_and_rename_crds
apply_config
