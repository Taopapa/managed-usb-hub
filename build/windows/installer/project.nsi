!include "MUI2.nsh"
!include "FileFunc.nsh"

;--------------------------------
;General

  ;Name and file
  Name "C2G USB Hub Manager"
  OutFile "..\..\bin\C2GUSBHubManager_Setup.exe"
  Unicode True

  ;Default installation folder
  InstallDir "$LOCALAPPDATA\C2G USB Hub Manager"

  ;Get installation folder from registry if available
  InstallDirRegKey HKCU "Software\C2G USB Hub Manager" ""

  ;Request application privileges for Windows Vista
  RequestExecutionLevel admin

;--------------------------------
;Interface Settings

  !define MUI_ABORTWARNING

;--------------------------------
;Pages

  !insertmacro MUI_PAGE_WELCOME
  !insertmacro MUI_PAGE_DIRECTORY
  !insertmacro MUI_PAGE_INSTFILES
  !insertmacro MUI_PAGE_FINISH

  !insertmacro MUI_UNPAGE_WELCOME
  !insertmacro MUI_UNPAGE_CONFIRM
  !insertmacro MUI_UNPAGE_INSTFILES
  !insertmacro MUI_UNPAGE_FINISH

;--------------------------------
;Languages

  !insertmacro MUI_LANGUAGE "English"

;--------------------------------
;Installer Sections

Section "C2G USB Hub Manager" SecDummy

  SetOutPath "$INSTDIR"

  ;ADD YOUR OWN FILES HERE...
  File "..\..\bin\C2G USB Hub Manager.exe"
  File "..\..\bin\muhcli.exe"
  File "..\wails.exe.manifest"

  ;Store installation folder
  WriteRegStr HKCU "Software\C2G USB Hub Manager" "" $INSTDIR

  ;Create uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Add to PATH feature has been removed
  ; Users can add it manually if needed

  ;Create Shortcuts
  CreateDirectory "$SMPROGRAMS\C2G USB Hub Manager"
  CreateShortcut "$SMPROGRAMS\C2G USB Hub Manager\C2G USB Hub Manager.lnk" "$INSTDIR\C2G USB Hub Manager.exe"
  CreateShortcut "$SMPROGRAMS\C2G USB Hub Manager\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
  
  ;Create Desktop Shortcut
  CreateShortcut "$DESKTOP\C2G USB Hub Manager.lnk" "$INSTDIR\C2G USB Hub Manager.exe"

SectionEnd

;--------------------------------
;Uninstaller Section

Section "Uninstall"

  ; Remove from PATH feature has been removed
  ; Users need to remove it manually

  Delete "$INSTDIR\C2G USB Hub Manager.exe"
  Delete "$INSTDIR\muhcli.exe"
  Delete "$INSTDIR\wails.exe.manifest"
  Delete "$INSTDIR\Uninstall.exe"

  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\C2G USB Hub Manager\C2G USB Hub Manager.lnk"
  Delete "$SMPROGRAMS\C2G USB Hub Manager\Uninstall.lnk"
  RMDir "$SMPROGRAMS\C2G USB Hub Manager"
  
  ;Remove Desktop Shortcut
  Delete "$DESKTOP\C2G USB Hub Manager.lnk"

  DeleteRegKey /ifempty HKCU "Software\C2G USB Hub Manager"

SectionEnd
