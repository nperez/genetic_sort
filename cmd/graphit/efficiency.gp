# Plot average InstructionsExecuted by generation
#
# Usage:
#   gnuplot -e "db='path/to/genetic_sort.db'; pop=1" efficiency.gp
#
# Defaults:
if (!exists("db")) db = "genetic_sort.db"
if (!exists("pop")) pop = 1

set terminal pngcairo size 900,500 enhanced

set title sprintf("Execution Efficiency Over Generations — Population %d", pop)
set xlabel "Generation"
set ylabel "Avg Instructions Executed"
set grid
set datafile separator "|"

plot '< sqlite3 '.db.' "SELECT u.generation, AVG(e.instructions_executed) FROM units u JOIN evaluations e ON e.unit_id = u.id WHERE u.population_id = '.pop.' GROUP BY u.generation ORDER BY u.generation"' \
    using 1:2 with linespoints title "Avg Instructions Executed"
