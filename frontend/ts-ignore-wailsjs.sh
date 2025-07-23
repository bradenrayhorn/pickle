#!/bin/bash

ts_nocheck_file() {
    local file_path="$1"
    
    {
        echo "// @ts-nocheck"
        grep -v "@ts-check" "$file_path"
    } > "${file_path}.tmp"

    mv "${file_path}.tmp" "$file_path"
}


export -f ts_nocheck_file
find wailsjs -type f -name "*.ts" -exec bash -c 'ts_nocheck_file "$0"' {} \;
find wailsjs -type f -name "*.js" -exec bash -c 'ts_nocheck_file "$0"' {} \;
