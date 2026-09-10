#!/bin/bash
# Reusable release policy functions for mostcomm CI

validate_stable_tag() {
    local tag=$1
    if [[ "$tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
        return 0
    else
        return 1
    fi
}

validate_publish_context() {
    local ref_type=$1
    local github_ref=$2
    if [[ "$ref_type" == "tag" && "$github_ref" == refs/tags/v* ]]; then
        return 0
    else
        return 1
    fi
}

map_release_mode() {
    local release_mode=$1
    case "$release_mode" in
        release-major) echo "major" ;;
        release-minor) echo "minor" ;;
        release-patch) echo "patch" ;;
        *) return 1 ;;
    esac
}
