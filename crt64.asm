; nasm -Dmain64=$start -Dtsize=0x$tsize -Ddsize=0x$dsize -Dbsize=0x$dsize -o crt64.bin crt64.asm 

            bits    16
            org     0x7c00

start:
            mov     ax, initlm

            ; hide cursor
            mov     ah, 0x01
            mov     cx, 0x2607
            int     0x10
            jmp     init

print:      lodsb
            test    al, al
            jz      doneprint
            stosb
            mov     al, 0x1f
            stosb
            jmp     print
doneprint:
            ret

init:
            ; disable interruptions
            cli
           
            ; In Real Mode, 16-bit effective addresses are zero-extended and
            ; added to a 16-bit segment-base address that is left-shifted
            ; four bits, producing a 20-bit linear address.

            ; initialize segment registers and stack pointer
            xor     ax, ax
            mov     ds, ax
            mov     ss, ax
            mov     sp, 0x7000

            ; CGA Video buffer address is 0xb8000
            mov     ax, 0xb800
            mov     es, ax

            ; clear screen
            xor     di, di
            mov     ax, 0x1f00
            mov     cx, 80*25
            rep     stosw

            ; print hellomsg
            xor     di, di
            mov     si, hellomsg
            call    print

            ; enable A20
            mov     al, 0xdd
            out     0x64, al

            ; print a20msg
            mov     di, 80*2*1
            mov     si, a20msg
            call    print

            ; disable IRQs
            mov     al, 0xff
            out     0xa1, al
            out     0x21, al

            ; print irqmsg
            mov     di, 80*2*2
            mov     si, irqmsg
            call    print

            ; initialize 64 bits
p4          equ     0x1000
p3          equ     0x2000
p2          equ     0x3000
p1          equ     0x4000

            ; zero page tables
            mov     di, p4
            mov     cx, 0x1000         ; 4k * 4 (stosd)
            xor     eax, eax
clearpt:    mov     [di], ax
            add     di, 4
            loop    clearpt

            ; setup page map level 4 table
            mov     ax, p3|3            ; PRESENT|WRITE
            mov     word [p4], ax

            ; setup page directory pointer table
            mov     ax, p2|3            ; PRESENT|WRITE
            mov     word [p3], ax

            ; setup page directory table
            mov     ax, p1|3            ; PRESENT|WRITE
            mov     word [p2], ax

            ; setup page table entries
            mov     cx, 512
            mov     di, p1
            mov     eax, 3              ; present|write
initpte:
            mov     dword [di], eax
            add     eax, 0x1000
            add     di, 8
            loop    initpte

            ; print ptemsg
            mov     di, 80*2*3
            mov     si, ptemsg
            call    print

            ; init empty idt
            lidt    [idt64]

            ; enable PAE
initlm:
            mov     eax, cr4
            or      eax, (1<<5)
            mov     cr4, eax

            ; init paging
            mov     eax, p4
            mov     cr3, eax

            ; set LME bit
            mov     ecx, 0xc0000080
            rdmsr
            or      eax, (1<<8)
            wrmsr

            ; enable paging and protection
            mov     eax, cr0
            or      eax, (1<<31)|1
            mov     cr0, eax

            ; init lgdt
            lgdt    [gdt64]
            jmp     dword 8:start64

            align   16
gdtnull     dq      0
gdtcode     dq      0x00209A0000000000
gdtdata     dq      0x0000920000000000
gdt64       dw      $-gdtnull-1
            dd      gdtnull

idt64       dw      0
            dd      0

hellomsg    db      "Welcome to PXE Network Boot Program", 0
a20msg      db      " - A20 Gate enabled", 0
irqmsg      db      " - IRQs disabled", 0
ptemsg      db      " - PTEs inited", 0

; --------------------------------------------------------------

            bits    64
            align   16
print64:    lodsb
            test    al, al
            jz      doneprint64
            stosb
            mov     al, 0x1f
            stosb
            jmp     print
doneprint64:
            ret

            align   16
start64:    mov     ax, 0x10
            mov     ds, ax
            mov     es, ax
            mov     fs, ax
            mov     gs, ax
            mov     ss, ax
            mov     rsp, 0x7000

            mov     rdi, 0xb8000+80*2*4
            mov     rsi, hello64msg
            call    print64

            mov     rsi, binary
            mov     rdi, taddr
            mov     rcx, tsize
            rep movsb

            mov     rdi, daddr
            mov     rcx, dsize
            rep movsb

            mov     rdi, baddr
            xor     rax, rax
            mov     rcx, bsize
            rep stosb

            call    main64

iloop64:    hlt
            jmp     iloop64

hello64msg  db      " - 64bits mode inited", 0

binary      equ     $
