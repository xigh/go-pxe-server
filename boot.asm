            bits 16
            org 0x7c00

start:
            ; hide cursor
            mov     ah, 0x01
            mov     cx, 0x2607
            int     0x10

            ; disable interruptions
            cli

            ; initialize segment registers
            xor     ax, ax
            mov     ds, ax
            mov     ax, 0xb800
            mov     es, ax

            ; clear screen
            xor     ax, ax
            mov     di, ax
            mov     cx, 80*25
            rep     stosw

            ; write hello world
            xor     di, di
            mov     si, message
print:            
            lodsb
            test    al, al
            jz      next
            stosb
            mov     al, 15
            stosb
            jmp     print

next:
            nop

            ; initialize 64 bits

iloop:      jmp     iloop

message     db      "Hello World !", 0

