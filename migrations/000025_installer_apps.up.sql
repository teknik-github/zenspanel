CREATE TABLE installer_apps (
  slug    VARCHAR(64)  NOT NULL,
  enabled TINYINT(1)   NOT NULL DEFAULT 1,
  PRIMARY KEY (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed the full catalog; each entry enabled by default.
INSERT INTO installer_apps (slug, enabled) VALUES
  ('wordpress',   1),
  ('joomla',      1),
  ('drupal',      1),
  ('prestashop',  1),
  ('codeigniter', 1),
  ('laravel',     1),
  ('html',        1);
