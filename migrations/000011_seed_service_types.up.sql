-- Seed: default service types for AMAUR occupational health platform

INSERT INTO service_types (name, category, description, default_duration_minutes, is_group_service, requires_clinical_record, is_active) VALUES
  -- Bienestar laboral (grupales)
  ('Pausa Activa', 'bienestar', 'Actividad grupal de movimiento y estiramiento en el lugar de trabajo', 30, true, false, true),
  ('Taller de Manejo del Estres', 'bienestar', 'Sesion grupal de tecnicas de relajacion y manejo del estres', 60, true, false, true),
  ('Charla de Salud Ocupacional', 'bienestar', 'Charla informativa sobre salud y bienestar en el trabajo', 45, true, false, true),
  ('Taller de Ergonomia', 'bienestar', 'Capacitacion sobre posturas correctas y uso del mobiliario', 60, true, false, true),
  ('Mindfulness Grupal', 'bienestar', 'Sesion grupal de atencion plena y meditacion', 30, true, false, true),

  -- Evaluacion individual
  ('Evaluacion Psicologica', 'evaluacion', 'Evaluacion individual del estado psicologico y bienestar mental', 60, false, true, true),
  ('Evaluacion Kinesica', 'evaluacion', 'Evaluacion fisioterapeutica individual', 60, false, true, true),
  ('Evaluacion de Riesgo Ergonomico', 'evaluacion', 'Evaluacion postural y de riesgo en puesto de trabajo', 45, false, true, true),
  ('Evaluacion Nutricional', 'evaluacion', 'Evaluacion del estado nutricional y habitos alimentarios', 45, false, true, true),

  -- Terapia individual
  ('Sesion de Psicoterapia', 'terapia', 'Sesion individual de psicoterapia', 50, false, true, true),
  ('Sesion de Kinesiterapia', 'terapia', 'Sesion individual de kinesiologia y rehabilitacion', 45, false, true, true),
  ('Sesion de Fonoaudiologia', 'terapia', 'Sesion individual de fonoaudiologia', 45, false, true, true),
  ('Consejeria Nutricional', 'terapia', 'Sesion de seguimiento y consejeria nutricional', 30, false, true, true),

  -- Capacitacion
  ('Capacitacion en Primeros Auxilios', 'capacitacion', 'Formacion en primeros auxilios basicos', 120, true, false, true),
  ('Taller de Liderazgo Saludable', 'capacitacion', 'Capacitacion para jefaturas en liderazgo y bienestar de equipos', 90, true, false, true);
