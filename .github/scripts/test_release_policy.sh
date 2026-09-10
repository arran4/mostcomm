#!/bin/bash
set -euo pipefail

source .github/scripts/release_policy.sh

echo "Running release policy non-mutating regression coverage tests"

FAIL=0

# Test validate_stable_tag
test_validate_tag() {
    local tag=$1
    local expected_fail=$2

    echo "Testing stable tag validation: $tag (Expected to fail: $expected_fail)"

    if validate_stable_tag "$tag"; then
        if [ "$expected_fail" = true ]; then
            echo "FAIL: Tag $tag should have been rejected."
            FAIL=1
        else
            echo "PASS"
        fi
    else
        if [ "$expected_fail" = true ]; then
            echo "PASS"
        else
            echo "FAIL: Tag $tag should have been accepted."
            FAIL=1
        fi
    fi
}

test_validate_tag "v1.0.0" false
test_validate_tag "v1.2.3" false
test_validate_tag "v0.1.2" false
test_validate_tag "v01.2.3" true
test_validate_tag "v1.02.3" true
test_validate_tag "v1.2.3-rc1" true
test_validate_tag "1.2.3" true
test_validate_tag "test" true

# Test validate_publish_context
test_publish_context() {
    local ref_type=$1
    local github_ref=$2
    local expected_fail=$3

    echo "Testing publish context: ref_type=$ref_type github_ref=$github_ref (Expected to fail: $expected_fail)"

    if validate_publish_context "$ref_type" "$github_ref"; then
        if [ "$expected_fail" = true ]; then
            echo "FAIL: Context should have been rejected."
            FAIL=1
        else
            echo "PASS"
        fi
    else
        if [ "$expected_fail" = true ]; then
            echo "PASS"
        else
            echo "FAIL: Context should have been accepted."
            FAIL=1
        fi
    fi
}

test_publish_context "tag" "refs/tags/v1.0.0" false
test_publish_context "branch" "refs/heads/main" true
test_publish_context "tag" "refs/tags/test" true
test_publish_context "branch" "refs/tags/v1.0.0" true

# Test map_release_mode
test_map_mode() {
    local mode=$1
    local expected_out=$2
    local expected_fail=$3

    echo "Testing map mode: $mode (Expected to fail: $expected_fail, Expected output: $expected_out)"

    set +e
    out=$(map_release_mode "$mode")
    rc=$?
    set -e

    if [ $rc -eq 0 ]; then
        if [ "$expected_fail" = true ]; then
            echo "FAIL: Mode $mode should have been rejected."
            FAIL=1
        elif [ "$out" != "$expected_out" ]; then
            echo "FAIL: Expected output $expected_out but got $out."
            FAIL=1
        else
            echo "PASS"
        fi
    else
        if [ "$expected_fail" = true ]; then
            echo "PASS"
        else
            echo "FAIL: Mode $mode should have been accepted."
            FAIL=1
        fi
    fi
}

test_map_mode "release-major" "major" false
test_map_mode "release-minor" "minor" false
test_map_mode "release-patch" "patch" false
test_map_mode "release-unsupported" "" true
test_map_mode "build" "" true

if [ $FAIL -eq 0 ]; then
    echo "All release policy tests passed."
    exit 0
else
    echo "Release policy tests failed."
    exit 1
fi
