#!/bin/bash
# get.s9s.dev - Simple redirect to GitHub raw content
# This file should be deployed to get.s9s.dev

curl -sSL https://raw.githubusercontent.com/jontk/s9s/main/scripts/install.sh | bash "$@"