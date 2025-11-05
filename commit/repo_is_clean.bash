#!/bin/env bash

# Copyright 2025 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

# This script is used by the GitHub Action which runs the tests and reformats
# the repository, ensuring that the checked commit was already formatted
# correctly.

set -e -o pipefail

if [[ "$(git status --porcelain)" ]]; then
    echo "The repository is not clean"
    exit 1
fi

echo "The repository is clean"