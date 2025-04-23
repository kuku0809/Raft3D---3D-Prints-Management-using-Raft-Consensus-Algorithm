#!/bin/bash

API_BASE="http://localhost:8080/api/v1"
CLUSTER_BASE="http://localhost:8080/cluster"

echo "=== Testing Raft3D API ==="
echo "Using base URL: $API_BASE"
echo

# 1. Cluster Info (before any operations)
echo "1. Getting initial cluster info:"
curl -s "$CLUSTER_BASE/leader" | jq
curl -s "$CLUSTER_BASE/state" | jq
echo

# 2. Printer Operations
echo "2. Testing Printer APIs:"

# 2.1 Create Printer
echo "Creating printer p1:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"p1","company":"Creality","model":"Ender 3"}' \
  "$API_BASE/printers" | jq

# 2.2 Create Duplicate Printer (should fail)
echo "Creating duplicate printer p1 (should fail):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"p1","company":"Creality","model":"Ender 3 V2"}' \
  "$API_BASE/printers" | jq

# 2.3 Create Printer with Missing Fields (should fail)
echo "Creating printer with missing fields (should fail):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"p2","company":"Prusa"}' \
  "$API_BASE/printers" | jq

# 2.4 List Printers
echo "Listing printers:"
curl -s "$API_BASE/printers" | jq
echo

# 3. Filament Operations
echo "3. Testing Filament APIs:"

# 3.1 Create Filament
echo "Creating filament f1:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"f1","type":"PLA","color":"red","total_weight_in_grams":1000,"remaining_weight_in_grams":1000}' \
  "$API_BASE/filaments" | jq

# 3.2 Create Another Filament
echo "Creating filament f2:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"f2","type":"ABS","color":"black","total_weight_in_grams":500,"remaining_weight_in_grams":500}' \
  "$API_BASE/filaments" | jq

# 3.3 List Filaments
echo "Listing filaments:"
curl -s "$API_BASE/filaments" | jq
echo

# 4. Print Job Operations
echo "4. Testing Print Job APIs:"

# 4.1 Create Print Job
echo "Creating print job j1:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"j1","printer_id":"p1","filament_id":"f1","filepath":"test.gcode","print_weight_in_grams":100}' \
  "$API_BASE/print_jobs" | jq

# 4.2 Create Print Job with Invalid Printer (should fail)
echo "Creating print job with invalid printer (should fail):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"j2","printer_id":"p99","filament_id":"f1","filepath":"test.gcode","print_weight_in_grams":100}' \
  "$API_BASE/print_jobs" | jq

# 4.3 Create Print Job with Invalid Filament (should fail)
echo "Creating print job with invalid filament (should fail):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"j3","printer_id":"p1","filament_id":"f99","filepath":"test.gcode","print_weight_in_grams":100}' \
  "$API_BASE/print_jobs" | jq

# 4.4 Create Print Job with Too Much Weight (should fail)
echo "Creating print job with too much weight (should fail):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"id":"j4","printer_id":"p1","filament_id":"f1","filepath":"test.gcode","print_weight_in_grams":2000}' \
  "$API_BASE/print_jobs" | jq

# 4.5 List Print Jobs
echo "Listing print jobs:"
curl -s "$API_BASE/print_jobs" | jq
echo

# 5. Print Job Status Updates
echo "5. Testing Print Job Status Updates:"

# 5.1 Update to Running
echo "Updating job j1 to Running:"
curl -s -X POST "$API_BASE/print_jobs/j1/status?status=running" | jq

# 5.2 Invalid Status Transition (should fail)
echo "Invalid status transition (should fail):"
curl -s -X POST "$API_BASE/print_jobs/j1/status?status=queued" | jq

# 5.3 Update to Done
echo "Updating job j1 to Done:"
curl -s -X POST "$API_BASE/print_jobs/j1/status?status=done" | jq

# 5.4 Verify Filament Weight Reduced
echo "Checking filament f1 remaining weight:"
curl -s "$API_BASE/filaments" | jq '.[] | select(.id == "f1")'
echo

# 6. Cluster Info (after operations)
echo "6. Getting final cluster info:"
curl -s "$CLUSTER_BASE/leader" | jq
curl -s "$CLUSTER_BASE/state" | jq
echo

echo "=== API Testing Complete ==="
