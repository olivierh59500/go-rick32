# Rick32 data formats

The original game renders a 640 × 480 window and advances gameplay every 40 ms. Its first level contains nine connected rooms. The attract recording contains 2,272 input samples at 25 Hz.

The embedded artwork consists of 58 blocks, 141 tiles of 8 × 8 pixels and 51 sprites of 32 × 21 pixels. A block contains sixteen tile indices. Tiles and sprites store one palette index per pixel. The sixteen colors use the original A1R5G5B5 palette.

The map contains 2,048 block indices. Room starts select a position in that map. Two connection records per room contain direction, outgoing row, target room and incoming row. Entity marks contain five bytes; entity definitions contain eight bytes; movement steps contain a duration and two signed displacements.

The text pages retain the original strings, line positions, tint and dwell duration. They move through cubic entrances and exits while individual characters oscillate on two independent sine waves.
