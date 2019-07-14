#!/usr/bin/env bash

# Copyright 2019 Google Inc. All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to writing, software distributed
# under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
# CONDITIONS OF ANY KIND, either express or implied.

# See the License for the specific language governing permissions and
# limitations under the License.

## Author: ptone@google.com (Preston Holmes)




set -e
set -o pipefail

# Print a usage message and exit.
usage(){
  local name
  name=$(basename "$0")

  cat >&2 <<-EOF
        ${name}

        You should have gcloud installed and configured with a project by running:

            $> gcloud auth login
            $> gcloud config set project [PROJECT_ID]

        This script performs the following steps: 
        - builds the current golang package as a plugin 
        - uploads the .so file to a GCS bucket
        - Triggers a harness function to pull and load that plugin

        The package must expose an httpHandler Func named "Handler" e.g.

            func Handler(w http.ResponseWriter, r *http.Request)

        You need to set two environment variables:


            PLUGIN_BUCKET - name of the GCS bucket (no gs:// scheme)
            HARNESS_URL - the base URL of the harness

example:

HARNESS_URL=https://dev-harness-azzzzzzz-uc.a.run.app

EOF

  exit 1
}

main(){

    if [[ -z "$HARNESS_URL" ]] ; then
        usage
    fi
    while true; do
        tmpfile=$(mktemp /tmp/build-XXXXXX)
        echo "building update"
        docker build -q -t build-tmp .
        id=$(docker create build-tmp)
        docker cp $id:/app $tmpfile
        docker rm -v $id
        # sleep 1
        # go build -o $tmpfile *.go

        echo "uploading"
        curl --header "Content-Type:application/octet-stream" \
        --data-binary @$tmpfile \
        $HARNESS_URL/_upload

        # rm $tmpfile

        echo "Ready - waiting for next change"
        inotifywait -q -r -e modify `pwd`
    done
}

main "$@"