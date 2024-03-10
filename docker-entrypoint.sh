#!/bin/sh

if [ "$1" = 'migrate_and_release' ]; then
    make database-check
    make database-drop
    make database-create
    make database-migration-up
    exec /app/rinha-de-backend-golang-2024-q1
elif [ "$1" = 'release' ]; then
    make database-check
    exec /app/rinha-de-backend-golang-2024-q1
fi
