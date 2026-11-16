param(
    [string]$Windres = 'windres'
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
Add-Type -AssemblyName System.Drawing

# Convert the transparent PNG into a multi-resolution Windows ICO.
# Small frames use 32-bit BGRA DIBs with AND masks; the 256px frame uses PNG.
$source = [System.Drawing.Bitmap]::FromFile((Join-Path $PWD 'assets/app.png'))
$sizes = @(16, 20, 24, 32, 40, 48, 64, 128, 256)
$frames = @()
try {
    foreach ($size in $sizes) {
        $bitmap = [System.Drawing.Bitmap]::new($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        $graphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
        $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $graphics.DrawImage($source, [System.Drawing.Rectangle]::new(0, 0, $size, $size))
        $graphics.Dispose()
        $stream = [System.IO.MemoryStream]::new()
        if ($size -eq 256) {
            $bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
        } else {
            $writer = [System.IO.BinaryWriter]::new($stream)
            $maskStride = [int]([Math]::Ceiling($size / 32.0) * 4)
            $writer.Write([int]40)
            $writer.Write([int]$size)
            $writer.Write([int]($size * 2))
            $writer.Write([uint16]1)
            $writer.Write([uint16]32)
            $writer.Write([int]0)
            $writer.Write([int]($size * $size * 4 + $maskStride * $size))
            1..4 | ForEach-Object { $writer.Write([int]0) }
            for ($y = $size - 1; $y -ge 0; $y--) {
                for ($x = 0; $x -lt $size; $x++) {
                    $pixel = $bitmap.GetPixel($x, $y)
                    $writer.Write([byte]$pixel.B)
                    $writer.Write([byte]$pixel.G)
                    $writer.Write([byte]$pixel.R)
                    $writer.Write([byte]$pixel.A)
                }
            }
            for ($y = $size - 1; $y -ge 0; $y--) {
                $mask = [byte[]]::new($maskStride)
                for ($x = 0; $x -lt $size; $x++) {
                    if ($bitmap.GetPixel($x, $y).A -eq 0) {
                        $index = [int][Math]::Floor($x / 8)
                        $mask[$index] = $mask[$index] -bor (128 -shr ($x % 8))
                    }
                }
                $writer.Write($mask)
            }
        }
        $frames += ,$stream.ToArray()
        $stream.Dispose()
        $bitmap.Dispose()
    }
} finally {
    $source.Dispose()
}
$file = [System.IO.File]::Create((Join-Path $PWD 'assets/app.ico'))
$output = [System.IO.BinaryWriter]::new($file)
try {
    $output.Write([uint16]0)
    $output.Write([uint16]1)
    $output.Write([uint16]$sizes.Count)
    $offset = 6 + 16 * $sizes.Count
    for ($i = 0; $i -lt $sizes.Count; $i++) {
        $dimension = $sizes[$i] % 256
        $output.Write([byte]$dimension)
        $output.Write([byte]$dimension)
        $output.Write([uint16]0)
        $output.Write([uint16]1)
        $output.Write([uint16]32)
        $output.Write([int]$frames[$i].Length)
        $output.Write([int]$offset)
        $offset += $frames[$i].Length
    }
    foreach ($frame in $frames) { $output.Write([byte[]]$frame) }
} finally {
    $output.Dispose()
}
& $Windres -i assets/windows.rc -O coff -o app_windows_amd64.syso
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
