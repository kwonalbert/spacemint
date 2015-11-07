#!/bin/sh

rm -f results.txt

for ((i=1;i<=$1;i++));
do
    go test -timeout 48h -index=$i>> results.txt
    du -hc /media/storage/Xi /media/storage/Xi-merkle | grep total >> results.txt
    echo $'\n' >> results.txt
done
