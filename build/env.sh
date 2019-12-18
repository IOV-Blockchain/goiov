#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
giovdir="$workspace/src/github.com/CarLiveChainCo"
if [ ! -L "$giovdir/goiov" ]; then
    mkdir -p "$giovdir"
    cd "$giovdir"
    ln -s ../../../../../. goiov
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$giovdir/goiov"
PWD="$giovdir/goiov"

# Launch the arguments with the configured environment.
exec "$@"
