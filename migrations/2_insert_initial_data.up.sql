INSERT INTO public.people (id,first_name,last_name,patronymic,age,gender,nationality,emails)
  VALUES (1,'Robert','Lee','',63,'male','US','{robert.lee@example.com}'),
   (2,'James','Anderson','',61,'male','US','{james.anderson@example.com}'),
   (3,'Maria','Martinez','',53,'female','ES','{maria.m@example.com}'),
   (4,'Patricia','Garcia','',58,'female','US','{patricia.g@example.com}'),
   (5,'Michael','Johnson','',57,'male','BG','{michael.j@example.com}'),
   (6,'Emily','Brown','',40,'female','US','{emily.brown@example.com}'),
   (7,'David','Wilson','',52,'male','US','{david.wilson@example.com}'),
   (8,'Sarah','Taylora','Janefer',41,'female','RU','{sarah.taylor@example.com}'),
   (9,'Ваня','Власов','',20,'male','RU','{vanvl@example.com}'),
   (10, 'Анна','Огнева','',25,'female','RU','{ann@example.com}');

ALTER SEQUENCE people_id_seq RESTART WITH 10;