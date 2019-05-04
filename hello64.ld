ENTRY(_start)
OUTPUT_FORMAT(elf64-x86-64)

SECTIONS
{
	. = 1M;
 	.text ALIGN(128) : { *(.text .rodata) }
	.data ALIGN(128) : { *(.data) }
 	.bss ALIGN(128) : { *(.bss) }
}
