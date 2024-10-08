base=$1
tests=$2
for i in `find $tests -maxdepth 1 -type d`; do
    echo "===== $base : $i ====="
    ./overlaycheck.sh $base $i
    echo
    echo
done