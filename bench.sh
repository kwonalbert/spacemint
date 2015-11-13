#!/bin/sh

rm -f results.txt

for ((i=1;i<=$1;i++));
do
    $GOPATH/bin/spacecoin -index=$i -file $2$i -mode gen >> results.txt
    $GOPATH/bin/spacecoin -index=$i -file $2$i -mode commit >> results.txt
    $GOPATH/bin/spacecoin -index=$i -file $2$i -mode check >> results.txt
    du -hc -B 1024 $2$i | grep total >> results.txt
    echo $'\n' >> results.txt
done
