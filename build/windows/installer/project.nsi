!include "MUI2.nsh"
!include "FileFunc.nsh"

;--------------------------------
;General

  ;Name and file
  Name "Managed USB Hub"
  OutFile "..\..\bin\ManagedUSBHub_Setup.exe"
  Unicode True

  ;Default installation folder
  InstallDir "$LOCALAPPDATA\Managed USB Hub"

  ;Get installation folder from registry if available
  InstallDirRegKey HKCU "Software\Managed USB Hub" ""

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

Section "Managed USB Hub" SecDummy

  SetOutPath "$INSTDIR"

  ;ADD YOUR OWN FILES HERE...
  File "..\..\bin\Managed USB Hub.exe"
  File "..\..\bin\hub-cli.exe"
  File "..\wails.exe.manifest"

  ;Store installation folder
  WriteRegStr HKCU "Software\Managed USB Hub" "" $INSTDIR

  ;Create uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ;Add to PATH
  EnVar::SetHKLM
  EnVar::AddValue "Path" "$INSTDIR"

  ;Create Shortcuts
  CreateDirectory "$SMPROGRAMS\Managed USB Hub"
  CreateShortcut "$SMPROGRAMS\Managed USB Hub\Managed USB Hub.lnk" "$INSTDIR\Managed USB Hub.exe"
  CreateShortcut "$SMPROGRAMS\Managed USB Hub\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
  
  ;Create Desktop Shortcut
  CreateShortcut "$DESKTOP\Managed USB Hub.lnk" "$INSTDIR\Managed USB Hub.exe"

SectionEnd

;--------------------------------
;Uninstaller Section

Section "Uninstall"

  ;Remove from PATH
  EnVar::SetHKLM
  EnVar::DeleteValue "Path" "$INSTDIR"

  Delete "$INSTDIR\Managed USB Hub.exe"
  Delete "$INSTDIR\hub-cli.exe"
  Delete "$INSTDIR\wails.exe.manifest"
  Delete "$INSTDIR\Uninstall.exe"

  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\Managed USB Hub\Managed USB Hub.lnk"
  Delete "$SMPROGRAMS\Managed USB Hub\Uninstall.lnk"
  RMDir "$SMPROGRAMS\Managed USB Hub"
  
  ;Remove Desktop Shortcut
  Delete "$DESKTOP\Managed USB Hub.lnk"

  DeleteRegKey /ifempty HKCU "Software\Managed USB Hub"

SectionEnd
