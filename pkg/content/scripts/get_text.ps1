
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes


# 1. Get the focused element
try {
    $focused = [System.Windows.Automation.AutomationElement]::FocusedElement
}
catch {
    exit
}

if ($null -eq $focused) { exit }

# 2. Walk up to find the top-level Window
$walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
$window = $focused
while ($null -ne $window -and $window.Current.ControlType -ne [System.Windows.Automation.ControlType]::Window) {
    try {
        $parent = $walker.GetParent($window)
        if ($null -eq $parent) { break }
        $window = $parent
    }
    catch { break }
}

# Target is the window if found, otherwise just the focused element
$target = if ($null -ne $window) { $window } else { $focused }

# Helper to extract text from an element
function Get-Text($el) {
    if ($null -eq $el) { return "" }
    try {
        if ($el.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)) {
            return $el.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern).DocumentRange.GetText(-1)
        }
        if ($el.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)) {
            return $el.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern).Current.Value
        }
        return $el.Current.Name
    }
    catch { return "" }
}

# 3. Strategy: Find the main content area
# Priority 1: Document Control (Browsers, Word)
$condDoc = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, [System.Windows.Automation.ControlType]::Document)
$docEl = $target.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condDoc)
$text = Get-Text $docEl

# Priority 2: Edit Control (Notepad, Editors) - if Document failed or empty
if ([string]::IsNullOrWhiteSpace($text)) {
    $condEdit = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, [System.Windows.Automation.ControlType]::Edit)
    # FindFirst might pick a small search bar, so let's try to find a few and pick the largest text?
    # For speed, let's just try the first one, or the focused one if it's an Edit.
    
    # If the focused element IS an Edit, use it (most likely what the user is using)
    if ($focused.Current.ControlType -eq [System.Windows.Automation.ControlType]::Edit) {
        $text = Get-Text $focused
    }
    else {
        $editEl = $target.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condEdit)
        $text = Get-Text $editEl
    }
}

# Priority 3: Fallback to focused element (if it wasn't covered above)
if ([string]::IsNullOrWhiteSpace($text)) {
    $text = Get-Text $focused
}

Write-Output $text
