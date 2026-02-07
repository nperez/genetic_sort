# Plot alive unit count per generation
#
# Usage:
#   gnuplot -e "db='path/to/genetic_sort.db'; pop=1" population_size.gp
#
# Defaults:
if (!exists("db")) db = "genetic_sort.db"
if (!exists("pop")) pop = 1

set terminal pngcairo size 900,500 enhanced

set title sprintf("Population Size Over Generations — Population %d", pop)
set xlabel "Generation"
set ylabel "Alive Units"
set grid
set datafile separator "|"

plot '< sqlite3 '.db.' "SELECT generation, COUNT(*) FROM units WHERE population_id = '.pop.' AND alive = 1 GROUP BY generation ORDER BY generation"' \
    using 1:2 with linespoints title "Alive Units"
