typedef char int8_t;
typedef unsigned char uint8_t;
typedef short int16_t;
typedef unsigned short uint16_t;
typedef int int32_t;
typedef unsigned int uint32_t;
typedef long long int64_t;
typedef unsigned long long uint64_t;

uint8_t * const cga = (uint8_t * const) 0xb8000;
uint32_t cursor = 6*2*80;

int cgaPrint(const char *msg) {
    uint16_t i = 0;
    for (;;) {
        uint8_t c = msg[i++];
        if (c == '\n') {
            uint32_t line = cursor / (80*2);
            cursor = (line + 1) * (80*2);
            continue;
        }
        if (c == 0)
            break;
        cga[cursor++] = c;
        cga[cursor++] = 0x1f;
    }
    return i;
}

void _start() {
    cgaPrint(u8"Hello from 64-bit C\n 1- compile to elf64-x86-64 with script loader\n 2- extract .text, .data and .bss\n 3- update boot loader to initialize sections\n 4- jump to entrypoint");
}
