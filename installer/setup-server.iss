; ============================================================
; Project Manager — инсталлятор СЕРВЕРА (Служба + Трей + Агент)
; ============================================================

[Setup]
AppName=Project Manager Server
AppVersion=1.0.0
AppPublisher=Your Company
DefaultDirName={autopf}\Project Manager
DefaultGroupName=Project Manager
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=ProjectManager-Server-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64compatible
SetupIconFile=assets\icon.ico
UninstallDisplayIcon={app}\project-manager-tray.exe

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Files]
; Служба
Source: "..\dist\project-manager.exe";       DestDir: "{app}"; Flags: ignoreversion
; Трей
Source: "..\dist\project-manager-tray.exe";  DestDir: "{app}"; Flags: ignoreversion
; Агент (нужен локально для открытия папок)
Source: "..\dist\pm-agent.exe";              DestDir: "{app}"; Flags: ignoreversion
; Статика и шаблоны
Source: "..\dist\web\*";                     DestDir: "{app}\web"; Flags: ignoreversion recursesubdirs createallsubdirs

[Dirs]
; Папка для логов — создаём заранее, с правом записи для всех
Name: "{app}\logs"; Permissions: users-modify

[Registry]
; Регистрация URL-протокола pm-agent://
Root: HKCR; Subkey: "pm-agent"; ValueType: string; ValueName: ""; ValueData: "URL:Project Manager Agent Protocol"; Flags: uninsdeletekey
Root: HKCR; Subkey: "pm-agent"; ValueType: string; ValueName: "URL Protocol"; ValueData: ""; Flags: uninsdeletekey
Root: HKCR; Subkey: "pm-agent\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\pm-agent.exe"" ""%1"""; Flags: uninsdeletekey

[Run]
; 1. Установка службы
Filename: "{app}\project-manager.exe"; Parameters: "-install"; \
    StatusMsg: "Установка службы..."; \
    Flags: runhidden waituntilterminated

; 2. Запуск Агента (чтобы кнопки открытия папок работали)
Filename: "{app}\pm-agent.exe"; \
    Flags: nowait runhidden

; 3. Запуск трея
Filename: "{app}\project-manager-tray.exe"; \
    Description: "Запустить Project Manager Server"; \
    Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/F /IM project-manager-tray.exe"; \
    Flags: runhidden waituntilterminated; \
    RunOnceId: "KillTray"
    
Filename: "taskkill.exe"; Parameters: "/F /IM pm-agent.exe"; \
    Flags: runhidden waituntilterminated; \
    RunOnceId: "KillAgent"
    
Filename: "{app}\project-manager.exe"; Parameters: "-remove"; \
    Flags: runhidden waituntilterminated; \
    RunOnceId: "RemoveService"

[UninstallDelete]
Type: filesandordirs; Name: "{app}\logs"
Type: files;          Name: "{app}\.env"

[Code]
var
  DBPage: TInputQueryWizardPage;
  HTTPPage: TInputQueryWizardPage;
  AdminPage: TInputQueryWizardPage;

procedure InitializeWizard();
begin
  // --- Страница 1: База данных ---
  DBPage := CreateInputQueryPage(wpSelectDir,
    'Подключение к базе данных',
    'Укажите параметры PostgreSQL',
    'Эти данные будут записаны в .env и использованы приложением.');
  DBPage.Add('Адрес сервера БД (host):', False);
  DBPage.Add('Порт БД:', False);
  DBPage.Add('Пользователь БД:', False);
  DBPage.Add('Пароль БД:', True);
  DBPage.Add('Имя базы данных:', False);

  DBPage.Values[0] := 'localhost';
  DBPage.Values[1] := '5432';
  DBPage.Values[2] := 'postgres';
  DBPage.Values[3] := '';
  DBPage.Values[4] := 'project_manager';

  // --- Страница 2: HTTP ---
  HTTPPage := CreateInputQueryPage(DBPage.ID,
    'HTTP-сервер',
    'Параметры веб-интерфейса',
    'Порт, на котором будет открываться веб-интерфейс.');
  HTTPPage.Add('HTTP-порт:', False);

  HTTPPage.Values[0] := '8080';

  // --- Страница 3: Администратор ---
  AdminPage := CreateInputQueryPage(HTTPPage.ID,
    'Администратор приложения',
    'Логин и пароль первого администратора',
    'Создаётся один раз, если таблица users пуста.');
  AdminPage.Add('Логин администратора:', False);
  AdminPage.Add('Пароль администратора:', True);

  AdminPage.Values[0] := 'admin';
  AdminPage.Values[1] := 'admin123';
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
      'HTTP_PORT=' + HTTPPage.Values[0] + #13#10 +
      'DB_HOST=' + DBPage.Values[0] + #13#10 +
      'DB_PORT=' + DBPage.Values[1] + #13#10 +
      'DB_USER=' + DBPage.Values[2] + #13#10 +
      'DB_PASSWORD=' + DBPage.Values[3] + #13#10 +
      'DB_NAME=' + DBPage.Values[4] + #13#10 +
      'DB_SSLMODE=disable' + #13#10 +
      'SESSION_SECRET=' + GenerateRandomSecret() + #13#10 +
      'BOOTSTRAP_ADMIN_USERNAME=' + AdminPage.Values[0] + #13#10 +
      'BOOTSTRAP_ADMIN_PASSWORD=' + AdminPage.Values[1] + #13#10 +
      'BOOTSTRAP_ADMIN_FULLNAME=Администратор' + #13#10;
    SaveStringToFile(EnvFile, EnvContent, False);
  end;
end;