BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "Projects" (
	"id"	INTEGER,
	"prj_id"	INTEGER NOT NULL,
	"description"	TEXT,
	PRIMARY KEY("id"),
	UNIQUE("prj_id")
);
CREATE TABLE IF NOT EXISTS "Reports" (
	"id"	INTEGER,
	"userid"	INTEGER NOT NULL,
	"prj_id"	INTEGER NOT NULL,
	"description"	TEXT,
	"mission_on"	BLOB DEFAULT FALSE,
	"mission_place"	TEXT,
	"mission_road"	BLOB DEFAULT FALSE,
	"start_time"	DATETIME,
	"end_time"	DATETIME,
	"disable"	BLOB DEFAULT FALSE,
	"timestamp"	DATETIME DEFAULT CURRENT_TIMESTAMP,
	"work_type"	INTEGER DEFAULT 0,
	PRIMARY KEY("id"),
	UNIQUE("userid","start_time")
);
CREATE TABLE IF NOT EXISTS "Users" (
	"id"	INTEGER,
	"username"	TEXT NOT NULL,
	"longname"	TEXT,
	"city"	TEXT NOT NULL,
	"email"	TEXT NOT NULL,
	"password"	TEXT NOT NULL DEFAULT 'popnr',
	"is_admin"	BLOB DEFAULT FALSE,
	"is_blocked"	BLOB DEFAULT FALSE,
	PRIMARY KEY("id"),
	UNIQUE("username")
);
CREATE TABLE IF NOT EXISTS "Vacations" (
	"id"	INTEGER,
	"userid"	INTEGER,
	"date_start"	date,
	"date_end"	date,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "WorkTypes" (
	"id"	INTEGER,
	"name"	TEXT NOT NULL,
	"calculate_hours"	BLOB DEFAULT 0,
	PRIMARY KEY("id" AUTOINCREMENT)
);

INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (0,'Офис','1');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (1,'Командировка','1');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (2,'Командировка (в дороге)','0');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (3,'Отгул в счет переработок',0);
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (4,'Удаленная работа','1');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (5,'Отгул за свой счет',0);
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (6,'Больничный',0);
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (7,'Офис (работа в выходной)','1');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (8,'Командировка (работа в выходной)','1');
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (9,'Командировка (выходной)',0);
INSERT INTO "WorkTypes" ("id","name","calculate_hours") VALUES (10,'Отпуск',0);
CREATE VIEW MainCalendar
as
SELECT
	report_date,
	name,
	hours,
	(strftime('%H:%M',start_time) || ' - ' || strftime('%H:%M',end_time) || ' (' || iif(prj_id=0,'Не учтено',prj_id) || '): ' || description) as description


 from MainExport

order by report_date DESC, name DESC;
CREATE VIEW MainExport
as
SELECT
	name,
	report_date,
	case cast (strftime('%w', start_time, 'localtime') as integer)
		when 0 then 'ВС'
		when 1 then 'ПН'
		when 2 then 'ВТ'
		when 3 then 'СР'
		when 4 then 'ЧТ'
		when 5 then 'ПТ'
	  else 'СБ' end as weekday,
	strftime('%H:%M',start_time) as start_time,
	strftime('%H:%M',end_time) as end_time,
	iif((select calculate_hours from WorkTypes where id=work_type), 
		case 
		when hours<9 then 0
		else 1 end, 
		0) as pause,

	iif((select calculate_hours from WorkTypes where id=work_type), iif(hours>8, hours-1,hours), 0) as hours,
	iif((select calculate_hours from WorkTypes where id=work_type), max(hours-9, 0), 0) as overwork,
	'' as spare1,
	'' as spare2,
	prj_id,
	work_type_text as place,
	description
from MainView;
CREATE VIEW MainView
as
SELECT
id,
	strftime('%Y-%m-%d', start_time, 'localtime') as report_date,
	work_type,
	(select name from WorkTypes where id=work_type) as work_type_text,
	userid,
	(Select longname from Users where id = userid) as name,
	start_time,
	end_time,
	(strftime('%s',end_time)-strftime('%s',start_time))/3600 as hours,
	prj_id,
	description,
	case cast (work_type as integer)
		when 1 then 1
		when 2 then 1
		when 8 then 1
		when 9 then 1
	  else 0 end as mission_on,
	mission_place
from Reports 
ORDER BY start_time DESC, name ASC;
CREATE VIEW ProjectsView
as
SELECT
	prj_id,
	(select description from Projects where Projects.prj_id=MainView.prj_id) as name,
	sum(hours) as hours


 from MainView
 group by prj_id;
CREATE VIEW VacationExport
as
with RECURSIVE dates(userid,report_date, date_end) as (
	select userid, date_start, date_end from Vacations
	union ALL
	select userid, date(report_date, "+1 day"), date_end
	FROM dates
	where report_date < date_end
)
select 
DISTINCT 
	(SELECT longname from users where id=userid) as name, 
	report_date,   
	case cast (strftime('%w', report_date, 'localtime') as integer)
		when 0 then 'ВС'
		when 1 then 'ПН'
		when 2 then 'ВТ'
		when 3 then 'СР'
		when 4 then 'ЧТ'
		when 5 then 'ПТ'
	  else 'СБ' end as weekday,
	"" as start_time,
	"" as end_time,
	"" as pause,
	"" as hours,
	"" as overwork,
	"" as spare1,
	"" as spare2,
	"" as prj_id,
	"" as place,
	"Отпуск" as description
from dates
ORDER BY userid DESC, report_date DESC;
CREATE VIEW VacationView
as
SELECT
	id,
	(Select longname from Users where id = userid) as name,
	date_start,
	date_end,
	cast ((julianday(date_end)-julianday(date_start)+1) as INTEGER) as days
	
from Vacations
ORDER by name, date_start;
CREATE VIEW Years
as
select strftime('%Y', report_date, 'localtime') as year from MainExport 
group by year
ORDER by year DESC;
COMMIT;
