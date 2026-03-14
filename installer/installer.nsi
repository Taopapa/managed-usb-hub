; ManagedUSBHub Installer Script
; Requires NSIS (http://nsis.sourceforge.net/)

!include "MUI2.nsh"

;--- Application Details ---
Name "Managed USB Hub Manager"
OutFile "..\ManagedUSBHub_Setup.exe"
InstallDir "$PROGRAMFILES64\ManagedUSBHub"
InstallDirRegKey HKCU "Software\ManagedUSBHub" ""
RequestExecutionLevel admin

;--- Interface Configuration ---
!define MUI_ABORTWARNING
!define MUI_ICON "..\build\windows\icon.ico" 
!define MUI_UNICON "..\build\windows\icon.ico"
!define MUI_HEADERIMAGE
; !define MUI_WELCOMEFINISHPAGE_BITMAP "..\build\windows\installer\welcome.bmp" ; Commented out as file is missing

;--- Pages ---
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

;--- Languages ---
!insertmacro MUI_LANGUAGE "English"
;!insertmacro MUI_LANGUAGE "SimpChinese"

;--- Installer Section ---
Section "MainSection" SecMain
    SetOutPath "$INSTDIR"
    
    ; Main GUI Application (Assumes wails build has run)
    ; The default Wails output name often contains spaces or matches the project name
    File "..\build\bin\Managed USB Hub.exe"
    
    ; Rename it on install if desired, or keep as is.
    ; If we want to rename it to match the shortcut target:
    ; File "/oname=ManagedUSBHub.exe" "..\build\bin\Managed USB Hub.exe"
    ; But easier to just update shortcuts to point to the real name.
    
    ; CLI Tool (Assumes built in root)
    File "..\hub-cli.exe" 
    
    ; Store installation folder
    WriteRegStr HKCU "Software\ManagedUSBHub" "" $INSTDIR
    
    ; Create Uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Add to Add/Remove Programs
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\ManagedUSBHub" "DisplayName" "Managed USB Hub Manager"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\ManagedUSBHub" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\ManagedUSBHub" "DisplayIcon" "$INSTDIR\Managed USB Hub.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\ManagedUSBHub" "Publisher" "Your Company"
    
    ; Create Shortcuts
    CreateDirectory "$SMPROGRAMS\Managed USB Hub"
    CreateShortCut "$SMPROGRAMS\Managed USB Hub\Managed USB Hub.lnk" "$INSTDIR\Managed USB Hub.exe"
    CreateShortCut "$SMPROGRAMS\Managed USB Hub\Hub CLI.lnk" "$INSTDIR\hub-cli.exe"
    CreateShortCut "$SMPROGRAMS\Managed USB Hub\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    CreateShortCut "$DESKTOP\Managed USB Hub.lnk" "$INSTDIR\Managed USB Hub.exe"
SectionEnd

;--- Uninstaller Section ---
Section "Uninstall"
    Delete "$INSTDIR\Managed USB Hub.exe"
    Delete "$INSTDIR\hub-cli.exe"
    Delete "$INSTDIR\uninstall.exe"

    Delete "$DESKTOP\Managed USB Hub.lnk"
    Delete "$SMPROGRAMS\Managed USB Hub\Managed USB Hub.lnk"
    Delete "$SMPROGRAMS\Managed USB Hub\Hub CLI.lnk"
    Delete "$SMPROGRAMS\Managed USB Hub\Uninstall.lnk"
    RMDir "$SMPROGRAMS\Managed USB Hub"

    RMDir "$INSTDIR"

    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\ManagedUSBHub"
    DeleteRegKey HKCU "Software\ManagedUSBHub"
SectionEnd
