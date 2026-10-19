#!/bin/sh
set -eu

export ODH_E2E_NAMESPACE="${ODH_E2E_NAMESPACE:-opendatahub-db-operator-e2e}"
export ODH_E2E_RELEASE="${ODH_E2E_RELEASE:-opendatahub-db-operator-e2e}"
export ODH_E2E_OPERATOR_IMAGE="${ODH_E2E_OPERATOR_IMAGE:-quay.io/rhoai/odh-db-operator-rhel9:latest}"

mode=test
case "${1:-}" in
  setup|test|teardown)
    mode="$1"
    shift
    ;;
esac

case "${mode}" in
  setup)
    /e2e/e2e-cleanup prepare-namespace --name "${ODH_E2E_NAMESPACE}"
    exec /e2e/helm upgrade --install "${ODH_E2E_RELEASE}" /e2e/chart \
      --namespace "${ODH_E2E_NAMESPACE}" --atomic --wait --timeout 5m \
      --set-string "operator.image.ref=${ODH_E2E_OPERATOR_IMAGE}" \
      --set-string operator.image.pullPolicy=Always \
      --set-string platform.type=SelfManagedRhoai \
      --set-string platform.version=3.6.0
    ;;
  teardown)
    cleanup_status=0
    run_cleanup() {
      "$@" || {
        status=$?
        if [ "${cleanup_status}" -eq 0 ]; then
          cleanup_status="${status}"
        fi
      }
    }
    run_cleanup /e2e/e2e-cleanup database-service
    run_cleanup /e2e/helm uninstall "${ODH_E2E_RELEASE}" --namespace "${ODH_E2E_NAMESPACE}" \
      --ignore-not-found --wait --timeout 5m
    run_cleanup /e2e/e2e-cleanup namespace --name "${ODH_E2E_NAMESPACE}"
    exit "${cleanup_status}"
    ;;
  test)
    mkdir -p /artifacts/e2e-results
    exec /e2e/gotestsum --raw-command --format standard-verbose \
      --junitfile /artifacts/e2e-results/xunit_report.xml -- \
      /e2e/test2json -t /e2e/e2e-tests -test.v "$@"
    ;;
esac
