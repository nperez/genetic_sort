#!/bin/bash
set -euo pipefail

usage() {
    echo "Usage: $0 [-d <database>] [-p <population_id>] [-o <output_dir>]"
    echo
    echo "  -d  Path to SQLite database (default: genetic_sort.db in current directory)"
    echo "  -p  Population ID (if omitted, lists available populations)"
    echo "  -o  Output directory for PNGs (default: ./plots)"
    exit 1
}

DB="genetic_sort.db"
POP=""
OUTDIR="plots"

while getopts "d:p:o:h" opt; do
    case "$opt" in
        d) DB="$OPTARG" ;;
        p) POP="$OPTARG" ;;
        o) OUTDIR="$OPTARG" ;;
        *) usage ;;
    esac
done

if [ ! -f "$DB" ]; then
    echo "Error: database '$DB' not found"
    echo "Use -d to specify the path to your database."
    exit 1
fi

if [ -z "$POP" ]; then
    echo "Available populations in $DB:"
    echo
    sqlite3 -header -column "$DB" \
        "SELECT id, unit_count, unit_mutation_chance, unit_lifespan FROM populations"
    echo
    echo "Re-run with -p <id> to generate plots."
    exit 0
fi

mkdir -p "$OUTDIR"

SCRIPTDIR="$(cd "$(dirname "$0")" && pwd)"

for gp in "$SCRIPTDIR"/*.gp; do
    name="$(basename "$gp" .gp)"
    echo "Generating ${name}.png ..."
    gnuplot -e "db='$DB'; pop=$POP" \
            -e "set output '$OUTDIR/${name}.png'" \
            "$gp"
done

echo "Done. PNGs written to $OUTDIR/"
