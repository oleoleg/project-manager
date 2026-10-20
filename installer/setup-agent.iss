; ============================================================
; Project Manager — инсталлятор КЛИЕНТА (Только Агент)
; ============================================================

[Setup]
AppName=Project Manager Agent
AppVersion=1.0.0
AppPublisher=Your Company
; Ставим в LOCALAPPDATA, чтобы не требовать прав администратора
DefaultDirName={localappdata}\Project Manager Agent
DefaultGroupName=Project Manager Agent
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=ProjectManager-Agent-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
ArchitecturesInstallIn64BitMode=x64compatible
SetupIconFile=assets\icon.ico

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Files]
Source: "..\dist\pm-agent.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Project Manager Agent"; Filename: "{app}\pm-agent.exe"
Name: "{group}\Удалить Project Manager Agent"; Filename: "{uninstallexe}"

[Registry]
; Регистрация URL-протокола в HKCU (без прав админа)
Root: HKCU; Subkey: "Software\Classes\pm-agent"; ValueType: string; ValueName: ""; ValueData: "URL:Project Manager Agent Protocol"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\pm-agent"; ValueType: string; ValueName: "URL Protocol"; ValueData: ""; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\Classes\pm-agent\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\pm-agent.exe"" ""%1"""; Flags: uninsdeletekey

[Run]
; Запуск Агента после установки
Filename: "{app}\pm-agent.exe"; Description: "Запустить Project Manager Agent"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/F /IM pm-agent.exe"; Flags: runhidden waituntilterminated

[Code]
var
  ServerPage: TInputQueryWizardPage;

procedure InitializeWizard();
begin
  // Страница настройки сервера
  ServerPage := CreateInputQueryPage(wpSelectDir,
    'Подключение к серверу',
    'Укажите адрес сервера Project Manager',
    'Этот адрес будет использоваться для перехода в веб-интерфейс.');
  ServerPage.Add('IP-адрес или домен сервера:', False);
  ServerPage.Add('Порт (HTTP):', False);

  ServerPage.Values[0] := 'localhost';
  ServerPage.Values[1] := '8080';
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ConfigContent: String;
  ConfigFile: String;
begin
  if CurStep = ssPostInstall then
  begin
    // Создаем папку в %APPDATA%\ProjectManager
    ForceDirectories(ExpandConstant('{userappdata}\ProjectManager'));
    
    // Формируем JSON для agent.config.json
    ConfigFile := ExpandConstant('{userappdata}\ProjectManager\agent.config.json');
    ConfigContent :=
      '{' + #13#10 +
      '  "autostart": true,' + #13#10 +
      '  "server_url": "http://' + ServerPage.Values[0] + ':' + ServerPage.Values[1] + '"' + #13#10 +
      '}';
      
    SaveStringToFile(ConfigFile, ConfigContent, False);
  end;
end;