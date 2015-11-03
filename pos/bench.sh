#!/bin/sh

rm -f results.txt

for ((i=1;i<=$1;i++));
do
    go test -index=$i >> results.txt
    du -hc Xi Xi-merkle | grep total >> results.txt
    echo $'\n' >> results.txt
done
