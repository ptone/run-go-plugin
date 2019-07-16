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
        - builds the current golang package as a Docker container
        - Extracts the binary from the container
        - Uploads the new binary to an existing harness function which runs it in a new process

        The server must respect the port environment variable and not be hardcoded to Cloud Run's 8080
        This is because the dev harness runs it at 6060 and proxies 8080

        You need to set the environment variable:


            HARNESS_URL - the base URL of the harness
        
        This is most easily done with

            export HARNESS_URL=\$(gcloud alpha run services describe dev-harness --region us-central1 --format='value(status.address.url)')

example:

HARNESS_URL=https://dev-harness-azzzzzzz-uc.a.run.app ${name}

EOF

  exit 1
}

main(){

    if [[ -z "$PROJECT" ]] ; then
        usage
    fi

    start=`date +%s`
    docker build -t gcr.io/$PROJECT/staging-test .
    docker push  gcr.io/$PROJECT/staging-test 
    gcloud alpha run deploy staging-test \
        --image gcr.io/$PROJECT/staging-test \
        --allow-unauthenticated \
        --platform managed \
        --region us-central1
    end=`date +%s`
    runtime=$((end-start))
    echo
    echo "------------ Done in $runtime seconds"
}

main "$@"