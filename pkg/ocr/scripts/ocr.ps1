# PowerShell script to perform OCR on an image file using Windows UWP APIs
param(
    [Parameter(Mandatory = $true)]
    [string]$ImagePath
)

# Add type definition for UWP OCR
$code = @"
using System;
using System.Threading.Tasks;
using Windows.Graphics.Imaging;
using Windows.Media.Ocr;
using Windows.Storage;
using Windows.Storage.Streams;

namespace Ideathon.OCR
{
    public class OcrHelper
    {
        public static string GetText(string imagePath)
        {
            return GetTextAsync(imagePath).GetAwaiter().GetResult();
        }

        private static async Task<string> GetTextAsync(string imagePath)
        {
            try
            {
                StorageFile file = await StorageFile.GetFileFromPathAsync(imagePath);
                using (IRandomAccessStream stream = await file.OpenAsync(FileAccessMode.Read))
                {
                    BitmapDecoder decoder = await BitmapDecoder.CreateAsync(stream);
                    using (SoftwareBitmap softwareBitmap = await decoder.GetSoftwareBitmapAsync())
                    {
                        OcrEngine engine = OcrEngine.TryCreateFromUserProfileLanguages();
                        if (engine == null)
                        {
                            return "Error: OCR not supported for current language.";
                        }

                        OcrResult result = await engine.RecognizeAsync(softwareBitmap);
                        return result.Text;
                    }
                }
            }
            catch (Exception ex)
            {
                return "Error: " + ex.Message;
            }
        }
    }
}
"@

# Load required assemblies
# We need to find the path to Windows.winmd or equivalent contracts
$assemblyPath = "C:\Windows\System32\WinMetadata\Windows.Foundation.UniversalApiContract.winmd"
if (-not (Test-Path $assemblyPath)) {
    # Fallback or alternative search might be needed, but this is standard on Win10/11
    Write-Output "Error: UniversalApiContract not found."
    exit 1
}

# Add-Type with referenced assemblies
try {
    Add-Type -TypeDefinition $code -ReferencedAssemblies $assemblyPath, "System.Runtime.WindowsRuntime" -Language CSharp
}
catch {
    Write-Output "Error compiling OCR helper: $_"
    exit 1
}

# Execute
[Ideathon.OCR.OcrHelper]::GetText($ImagePath)
