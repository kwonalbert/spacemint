#!/bin/sh

rm -f results.txt

for ((i=22;i<=$1;i++));
do
    rm -f /media/storage/Xi$i
    $GOPATH/bin/spacecoin -index=$i -file /media/storage/Xi$i -mode gen >> results.txt
    $GOPATH/bin/spacecoin -index=$i -file /media/storage/Xi$i -mode commit >> results.txt
    du -hc -B 1024 /media/storage/Xi$i | grep total >> results.txt
    echo $'\n' >> results.txt
done
