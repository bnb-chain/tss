#!/usr/bin/env bash

kill -9 $(ps aux | grep "tss" | grep -v "grep" | grep -v "___tss" | grep -v "_atsserver" | awk '{print $2}')
