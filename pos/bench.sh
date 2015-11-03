#!/bin/sh

rm results.txt

for ((i=1;i<=$1;i++));
do
    go test -v -run=TestEmpty -index=$i >> results.txt
    du -hc Xi | grep total >> results.txt
    echo $'\n' >> results.txt
done
