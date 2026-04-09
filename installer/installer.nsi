; ManagedUSBHub Installer Script
; Requires NSIS (http://nsis.sourceforge.net/)

!include "MUI2.nsh"

;--- Application Details ---
Name "C2G USB Hub Manager"
OutFile "..\C2GUSBHubManager_Setup.exe"
InstallDir "$PROGRAMFILES64\C2G USB Hub Manager"
InstallDirRegKey HKCU "Software\C2G USB Hub Manager" ""
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
    File "..\build\bin\C2G USB Hub Manager.exe"
    
    ; Rename it on install if desired, or keep as is.
    ; If we want to rename it to match the shortcut target:
    ; File "/oname=C2GUSBHubManager.exe" "..\build\bin\C2G USB Hub Manager.exe"
    ; But easier to just update shortcuts to point to the real name.
    
    ; CLI Tool (Assumes built in root)
    File "..\muhcli.exe" 
    
    ; Store installation folder
    WriteRegStr HKCU "Software\C2G USB Hub Manager" "" $INSTDIR
    
    ; Create Uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Add to Add/Remove Programs
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\C2GUSBHubManager" "DisplayName" "C2G USB Hub Manager"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\C2GUSBHubManager" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\C2GUSBHubManager" "DisplayIcon" "$INSTDIR\C2G USB Hub Manager.exe"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\C2GUSBHubManager" "Publisher" "Your Company"
    
    ; Create Shortcuts
    CreateDirectory "$SMPROGRAMS\C2G USB Hub Manager"
    CreateShortCut "$SMPROGRAMS\C2G USB Hub Manager\C2G USB Hub Manager.lnk" "$INSTDIR\C2G USB Hub Manager.exe"
    CreateShortCut "$SMPROGRAMS\C2G USB Hub Manager\Hub CLI.lnk" "$INSTDIR\muhcli.exe"
    CreateShortCut "$SMPROGRAMS\C2G USB Hub Manager\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    CreateShortCut "$DESKTOP\C2G USB Hub Manager.lnk" "$INSTDIR\C2G USB Hub Manager.exe"
SectionEnd

;--- Uninstaller Section ---
Section "Uninstall"
    Delete "$INSTDIR\C2G USB Hub Manager.exe"
    Delete "$INSTDIR\muhcli.exe"
    Delete "$INSTDIR\uninstall.exe"

    Delete "$DESKTOP\C2G USB Hub Manager.lnk"
    Delete "$SMPROGRAMS\C2G USB Hub Manager\C2G USB Hub Manager.lnk"
    Delete "$SMPROGRAMS\C2G USB Hub Manager\Hub CLI.lnk"
    Delete "$SMPROGRAMS\C2G USB Hub Manager\Uninstall.lnk"
    RMDir "$SMPROGRAMS\C2G USB Hub Manager"

    RMDir "$INSTDIR"

    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\C2GUSBHubManager"
    DeleteRegKey HKCU "Software\C2G USB Hub Manager"
SectionEnd
