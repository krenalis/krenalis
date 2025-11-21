-- TODO(Gianluca): fill this file with update queries.

UPDATE members SET email = 'acme@meergo.com' WHERE email = 'acme@open2b.com';
UPDATE members SET email = 'docker@meergo.com' WHERE email = 'docker@open2b.com';

--                 Password:   meergo-password.
UPDATE members SET password = '$2a$10$dGlVroo3N23Vn99edSPe..xo1hhKzGLYafIjFQjazu3faeFizvW7m' WHERE email = 'acme@meergo.com';
UPDATE members SET password = '$2a$10$dGlVroo3N23Vn99edSPe..xo1hhKzGLYafIjFQjazu3faeFizvW7m' WHERE email = 'docker@meergo.com';
