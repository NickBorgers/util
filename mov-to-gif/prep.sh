#!/bin/sh

mkdir -p ./content

cp `ls ./*.mov| head -1` ./content/input
