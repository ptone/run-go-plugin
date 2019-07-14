# run-go-plugin

- run: [Cloud Run][]
- go: [Go language][]
- plugin: [Go plugins][]

## Introduction

This is an experiment in using [Go Plugins][] to allow server side "hot reloading" of code in development. 

[Cloud Run][] allows deployment of serverless functions in containers. In general a bias for fastest write-build-test loop would favor:

local env > local container rebuild/run > cloud container rebuild/run

However there may be conditions in the Cloud runtime that are difficult to recreate such as running as a specific service account for which you can not get a key to emulate locally. Or private VPC connections that are not available locally.

The idea in the project is to rebuild your serverless binary as a Go plugin, and ship it to a running Cloud Run Service that loads the plugin and serves the new handler.

## Try it

Configure your cloud project

        gcloud config set [your-project]
        PROJECT=$(gcloud config list --format 'value(core.project)')

make a bucket to store the plugins

    export PLUGIN_BUCKET=$PROJECT-run-plugin
    gsutil mb gs://$PLUGIN_BUCKET


build and deploy the harness

    cd plugin-harness
    gcloud builds submit --tag gcr.io/$PROJECT/go-harness

    gcloud alpha run deploy dev-harness \
    --image gcr.io/$PROJECT/go-harness \
    --allow-unauthenticated \
    --platform managed \
    --region us-central1

    export HARNESS_URL=$(gcloud alpha run services describe dev-harness --region us-central1 --format='value(status.address.url)')

Move to the sample developement example. this example is pretty much a standard cloud run hello-world. To work with the plugin harness, it most export an http handler named "Handler".

    cd ../std-run

    bash update.sh

    curl $HARNESS_URL

You should see a simple hello world

Now edit main.go and change the hello world to something else and update

    bash update.sh

    curl $HARNESS_URL

When you are done iterating, you can build and deploy the main service

    gcloud builds submit --tag gcr.io/$PROJECT/hello-prod

    gcloud alpha run deploy hello-prod \
    --image gcr.io/$PROJECT/hello-prod \
    --allow-unauthenticated \
    --platform managed \
    --region us-central1

## Caveats and Alternative strategies

This is just an experiment, and does not represent a finished or ideal state. This is only suitable for dev, maybe staging envs.

 - Go plugins can not be unloaded, so there is a built in memory leak on reloading, the container can be killed at the _die endpoint
 - There is currently no security model, anyone could hit the _reload or _die endpoints on the harness
 - I was not able to find a way to assign a new function to a handler in a live server, so the current technique restarts the server

The goal is speed, and when speed is the goal, you have to look at tradeoffs. The plugin for the demo hello world is ~12mb. If you are on a cellular or severly asymettrical connection this could take a while to upload. A more complex function could be much larger. Alternatives might be:

- ship just the source code, build in container
    - This has the advantage of sending only small source code files up to the cloud
    - if your build process has to fetch a bunch of dependencies each time that could be slow
- ship a full server binary instead of just the handler
    - this means shipping more bytes, but probably not many, and since I couldn't figure out how to avoid server restart anyway could be simpler
- develop on a VM with same identity and network access as Cloud Run
    - this is the most cumbersome and brittle, but is going to have the best performance profile. It is basically all the advantages of "local development" but in the runtime context close to the production runtime.

Not an official Google product.

[Cloud Run]: https://cloud.google.com/run/
[Go language]: https://golang.org
[Go plugins]: https://golang.org/pkg/plugin/