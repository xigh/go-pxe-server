#!/bin/bash

rm -rf hello64 hello64.o
gcc -std=gnu11 \
    -fno-unwind-tables \
    -fno-asynchronous-unwind-tables \
    -ffreestanding \
    -mno-sse -mno-mmx -mno-3dnow -mno-80387 \
    -fpic \
    -Qn \
    -Wall -Wextra \
    -nostdlib \
    -c hello64.c \
    -o hello64.o
ld -o hello64 -T hello64.ld hello64.o
ls -l hello64
strip -s hello64
strip --remove-section=.note.gnu.property hello64
strip --remove-section=.comment hello64
# 
# objdump -x hello64
objdump -D hello64
# hexdump -v -C hello64
ls -l hello64

tsize=`objdump -h hello64 | grep .text | awk '{ print $3 }'`;
taddr=`objdump -h hello64 | grep .text | awk '{ print $4 }'`;
start=`objdump -f hello64 | grep 'start address' | awk '{print $3}'`;
dsize=`objdump -h hello64 | grep .data | awk '{ print $3 }'`;
daddr=`objdump -h hello64 | grep .data | awk '{ print $4 }'`;
bsize=`objdump -h hello64 | grep .bss | awk '{ print $3 }'`;
baddr=`objdump -h hello64 | grep .bss | awk '{ print $4 }'`;

# gen hello64.text
objdump -h hello64 |
    grep .text |
    awk '{print "dd if=hello64 of=hello64.text bs=1 count=$[0x" $3 "] skip=$[0x" $6 "]"}' |
    bash

# gen hello64.data
objdump -h hello64 |
    grep .data |
    awk '{print "dd if=hello64 of=hello64.data bs=1 count=$[0x" $3 "] skip=$[0x" $6 "]"}' |
    bash

echo "start=$start text=0x$tsize data=0x$dsize bss=0x$dsize"
nasm -Dmain64=$start \
    -Dtaddr=0x$taddr -Dtsize=0x$tsize \
    -Ddaddr=0x$daddr -Ddsize=0x$dsize \
    -Dbaddr=0x$baddr -Dbsize=0x$bsize \
    -o crt64.bin crt64.asm 
cat crt64.bin hello64.text hello64.data > boot.bin
ls -l boot.bin
objdump -h hello64
