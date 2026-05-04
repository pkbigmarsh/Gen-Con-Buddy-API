#!/bin/bash
echo "Fetching gencon events xlsx from ${GENCON_EVENT_URL}"
current_event_path=${RAILWAY_VOLUME_MOUNT_PATH}/$(date +"%Y")
event_filename=$(date +"%Y%m%d_%H")_events.xlsx
mkdir -p $current_event_path


curl -o ${current_event_path}/${event_filename} -L -O ${GENCON_EVENT_URL}
if [ $? -ne 0 ]; then
    echo "Event curl failed $?"
    exit $?
fi

bin/gcb data update --filepath "${current_event_path}/${event_filename}"


# https://www.gencon.com/downloads/events.xlsx