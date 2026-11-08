/* Match the flash and main SRAM regions in the tested C linker.ld.
 * TinyGo needs its own startup symbols and stack/heap sections from arm.ld. */
MEMORY
{
    FLASH_TEXT (rx) : ORIGIN = 0x08000000, LENGTH = 2M
    RAM (xrw) : ORIGIN = 0x20000000, LENGTH = 192K
}
_stack_size = 4K;
INCLUDE "targets/arm.ld"
