; ============================================================
; Project Manager — инсталлятор
; ============================================================

[Setup]
AppName=Project Manager
AppVersion=1.0.0
AppPublisher=Your Company
AppPublisherURL=https://example.com
DefaultDirName={autopf}\Project Manager
DefaultGroupName=Project Manager
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=ProjectManager-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64
SetupIconFile=assets\icon.ico
UninstallDisplayIcon={app}\project-manager-tray.exe

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Tasks]
Name: "autostarttray"; Description: "Запускать трей при входе пользователя в систему"; GroupDescription: "Автозагрузка:"

[Files]
; Служба
Source: "..\dist\project-manager.exe";       DestDir: "{app}"; Flags: ignoreversion
; Трей
Source: "..\dist\project-manager-tray.exe";  DestDir: "{app}"; Flags: ignoreversion
; Статика и шаблоны
Source: "..\dist\web\*";                     DestDir: "{app}\web"; Flags: ignoreversion recursesubdirs createallsubdirs

[Run]
; 1. Установка службы
Filename: "{app}\project-manager.exe"; Parameters: "-install"; \
    StatusMsg: "Установка службы..."; \
    Flags: runhidden waituntilterminated

; 2. Запуск трея
Filename: "{app}\project-manager-tray.exe"; \
    Description: "Запустить Project Manager"; \
    Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{app}\project-manager.exe"; Parameters: "-remove"; \
    Flags: runhidden waituntilterminated; \
    RunOnceId: "RemoveService"

[UninstallDelete]
Type: filesandordirs; Name: "{app}\logs"
Type: files;          Name: "{app}\.env"

[Code]
var
  ConfigPage: TInputQueryWizardPage;
  ServicePage: TInputOptionWizardPage;

procedure InitializeWizard();
begin
  ConfigPage := CreateInputQueryPage(wpSelectTasks,
    'Подключение к базе данных',
    'Укажите параметры PostgreSQL',
    'Эти данные будут записаны в .env и использованы приложением.');
  ConfigPage.Add('Адрес сервера БД (host):', False);
  ConfigPage.Add('Порт БД (по умолчанию 5432):', False);
  ConfigPage.Add('Пользователь БД:', False);
  ConfigPage.Add('Пароль БД:', True);
  ConfigPage.Add('Имя базы данных:', False);
  ConfigPage.Add('Порт HTTP-сервера:', False);
  ConfigPage.Add('Логин администратора приложения:', False);
  ConfigPage.Add('Пароль администратора приложения:', True);

  ConfigPage.Values[0] := 'localhost';
  ConfigPage.Values[1] := '5432';
  ConfigPage.Values[2] := 'postgres';
  ConfigPage.Values[4] := 'project_manager';
  ConfigPage.Values[5] := '8080';
  ConfigPage.Values[6] := 'admin';
  ConfigPage.Values[7] := 'admin123';
end;

function GenerateRandomSecret(): String;
var
  I: Integer;
  Chars: String;
begin
  Chars := 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  Result := '';
  for I := 1 to 48 do
    Result := Result + Chars[Random(Length(Chars)) + 1];
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  EnvContent: String;
  EnvFile: String;
begin
  if CurStep = ssPostInstall then
  begin
    EnvFile := ExpandConstant('{app}\.env');
    EnvContent :=
      'APP_ENV=production' + #13#10 +
      'HTTP_PORT=' + ConfigPage.Values[5] + #13#10 +
      'DB_HOST=' + ConfigPage.Values[0] + #13#10 +
      'DB_PORT=' + ConfigPage.Values[1] + #13#10 +
      'DB_USER=' + ConfigPage.Values[2] + #13#10 +
      'DB_PASSWORD=' + ConfigPage.Values[3] + #13#10 +
      'DB_NAME=' + ConfigPage.Values[4] + #13#10 +
      'DB_SSLMODE=disable' + #13#10 +
      'SESSION_SECRET=' + GenerateRandomSecret() + #13#10 +
      'BOOTSTRAP_ADMIN_USERNAME=' + ConfigPage.Values[6] + #13#10 +
      'BOOTSTRAP_ADMIN_PASSWORD=' + ConfigPage.Values[7] + #13#10 +
      'BOOTSTRAP_ADMIN_FULLNAME=Администратор' + #13#10;
    SaveStringToFile(EnvFile, EnvContent, False);
  end;
end;