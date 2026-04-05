#!/bin/bash
# Update release.sh to use jamon8888 fork
FILE="/mnt/c/Users/NMarchitecte/Documents/msgvault/scripts/release.sh"
sed -i 's|wesm/msgvault|jamon8888/msgvault|g' "$FILE"