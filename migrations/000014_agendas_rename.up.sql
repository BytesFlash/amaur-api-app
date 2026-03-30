-- Rename visits to agendas at DB level.
-- Keep a compatibility view named visits so existing SQL can continue working.

ALTER TABLE visits RENAME TO agendas;
ALTER TABLE visit_workers RENAME TO agenda_workers;

ALTER INDEX idx_visits_company RENAME TO idx_agendas_company;
ALTER INDEX idx_visits_scheduled_date RENAME TO idx_agendas_scheduled_date;
ALTER INDEX idx_visits_status RENAME TO idx_agendas_status;
ALTER INDEX idx_visits_upcoming RENAME TO idx_agendas_upcoming;

CREATE VIEW visits AS SELECT * FROM agendas;
CREATE VIEW visit_workers AS SELECT * FROM agenda_workers;
