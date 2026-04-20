#!/usr/bin/env python3
"""Inject bgg_eval.csv and data.csv (BGM rows) as JSON into viewer_template.html."""

import csv
import json
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT  = os.path.join(SCRIPT_DIR, '..', '..')

BGM_FIELDS = [
    'Game ID', 'Title', 'Short Description', 'Event Type',
    'Game System', 'Rules Edition', 'Minimum Players', 'Maximum Players',
    'Age Required', 'Experience Required', 'Start Date & Time', 'Duration',
    'Location', 'Room Name', 'Table Number', 'GM Names',
    'Website', 'Cost $', 'Tickets Available', 'Tournament?',
]

def read_eval_csv(path):
    rows = []
    with open(path, newline='', encoding='utf-8') as f:
        for row in csv.DictReader(f):
            rows.append(dict(row))
    return rows

def read_events_index(path):
    """Returns dict: 'GameSystem||RulesEdition' -> [event, ...]"""
    index = {}
    with open(path, newline='', encoding='cp1252') as f:
        for row in csv.DictReader(f):
            if not row.get('Event Type', '').startswith('BGM'):
                continue
            event = {k: row.get(k, '').strip() for k in BGM_FIELDS}
            key = row.get('Game System', '').strip() + '||' + row.get('Rules Edition', '').strip()
            index.setdefault(key, []).append(event)
    return index

def main():
    eval_path     = os.path.join(REPO_ROOT, 'bgg_eval.csv')
    events_path   = os.path.join(REPO_ROOT, 'data.csv')
    template_path = os.path.join(SCRIPT_DIR, 'viewer_template.html')
    output_path   = os.path.join(REPO_ROOT, 'bgg_eval_viewer.html')

    print('Reading eval CSV...')
    eval_data = read_eval_csv(eval_path)
    print(f'  {len(eval_data)} combos')

    print('Reading Gen Con events CSV...')
    events_index = read_events_index(events_path)
    total = sum(len(v) for v in events_index.values())
    print(f'  {total} BGM events across {len(events_index)} combos')

    with open(template_path, encoding='utf-8') as f:
        html = f.read()

    html = html.replace('/*EVAL_DATA_PLACEHOLDER*/',   json.dumps(eval_data))
    html = html.replace('/*EVENTS_DATA_PLACEHOLDER*/', json.dumps(events_index))

    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(html)

    size_kb = os.path.getsize(output_path) // 1024
    print(f'Done. {output_path} ({size_kb} KB)')

if __name__ == '__main__':
    main()
