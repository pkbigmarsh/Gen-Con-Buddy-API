#!/bin/bash
# Run from this folder

run_update() {
    echo -e "#\n##\n#### --- ${1} --- ###\n##\n#" 
    ./gcb_test data update \
        --os_address "http://localhost:9200" \
        --local_file "./${1}"

    if [ $? -ne 0 ]; then
        echo -e "#\n##\n#### --- ${1} failed --- ###\n##\n#"
        return 1
    fi

    return 0
}

go build -ldflags="-w -s" -o ./gcb_test ../../

echo -e "#\n##\n#### --- Data Init --- ###\n##\n#"
./gcb_test data init -c \
    --os_address "http://localhost:9200" \
    --filepath "./00_10-initial-events.csv"

if [ $? -ne 0 ]; then
    echo -e "#\n##\n#### --- Data Init failed --- ###\n##\n#"
fi

test_files=("01_no-changes.csv" "02_add-5-events.csv" "03_add-1-modify-5.csv" "04_delete-10-add-5-modify-1.csv" "05_delete-7.csv")
for test in "${test_files[@]}"; do
    run_update $test
    if [ $? -ne 0 ]; then
        echo -e "#\n##\n#### --- Aborting early --- ###\n##\n#"
        break
    fi
done

rm ./gcb_test