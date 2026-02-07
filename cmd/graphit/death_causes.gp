# Histogram of death causes (tombstone reasons)
#
# Usage:
#   gnuplot -e "db='path/to/genetic_sort.db'; pop=1" death_causes.gp
#
# Reason codes:
#   1 = FailedMachineRun
#   2 = FailedSetFidelity
#   3 = FailedSortedness
#   4 = FailedInstructionCount
#   5 = FailedInstructionsExecuted
#
# Defaults:
if (!exists("db")) db = "genetic_sort.db"
if (!exists("pop")) pop = 1

set terminal pngcairo size 900,500 enhanced

set title sprintf("Death Causes — Population %d", pop)
set xlabel "Reason"
set ylabel "Count"
set style data histogram
set style fill solid 0.8
set boxwidth 0.6
set grid ytics
set datafile separator "|"

set xtics ("MachineRun" 1, "SetFidelity" 2, "Sortedness" 3, "InstrCount" 4, "InstrExec" 5)

plot '< sqlite3 '.db.' "SELECT t.reason, COUNT(*) FROM tombstones t JOIN units u ON u.id = t.unit_id WHERE u.population_id = '.pop.' GROUP BY t.reason ORDER BY t.reason"' \
    using 1:2 with boxes title "Deaths" lc rgb "#cc4444"
