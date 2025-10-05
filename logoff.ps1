# PowerShell script to logoff current user
# This script is called when time limit is exceeded

Write-Output "Executing logoff..."
shutdown /l
