#define MyAppName "DSA"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "DSA"

[Setup]
AppId={{4E4A0A2A-5E57-4D44-A1A8-8E9A4E2A4B10}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={localappdata}\Programs\DSA
DefaultGroupName=DSA
DisableProgramGroupPage=yes
OutputDir=output
OutputBaseFilename=DSA-Setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest

[Languages]
Name: "korean"; MessagesFile: "compiler:Languages\Korean.isl"

[Tasks]
Name: "desktopicon"; Description: "바탕 화면 아이콘 만들기"; GroupDescription: "추가 작업:"
Name: "startup"; Description: "Windows 시작 시 DSA 자동 실행"; GroupDescription: "추가 작업:"

[Files]
Source: "..\dist\dsa.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\.env.example"; DestDir: "{app}"; DestName: ".env.example"; Flags: ignoreversion

[Dirs]
Name: "{app}\logs"

[Icons]
Name: "{autoprograms}\DSA"; Filename: "{app}\dsa.exe"
Name: "{autodesktop}\DSA"; Filename: "{app}\dsa.exe"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "DSA"; ValueData: """{app}\dsa.exe"" gui"; Tasks: startup; Flags: uninsdeletevalue

[Run]
Filename: "{app}\dsa.exe"; Parameters: "gui"; Description: "DSA 설정 창 실행"; Flags: nowait postinstall skipifsilent
