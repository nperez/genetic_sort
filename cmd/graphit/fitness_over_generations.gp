# Plot average SetFidelity and Sortedness by generation
#
# Usage:
#   gnuplot -e "db='path/to/genetic_sort.db'; pop=1" fitness_over_generations.gp
#
# Defaults:
if (!exists("db")) db = "genetic_sort.db"
if (!exists("pop")) pop = 1

set terminal pngcairo size 900,500 enhanced

set title sprintf("Fitness Over Generations — Population %d", pop)
set xlabel "Generation"
set ylabel "Score (0–100)"
set yrange [0:100]
set grid
set key top left
set datafile separator "|"

plot '< sqlite3 '.db.' "SELECT u.generation, AVG(e.set_fidelity), AVG(e.sortedness) FROM units u JOIN evaluations e ON e.unit_id = u.id WHERE u.population_id = '.pop.' GROUP BY u.generation ORDER BY u.generation"' \
    using 1:2 with linespoints title "Avg Set Fidelity", \
    '' using 1:3 with linespoints title "Avg Sortedness"
