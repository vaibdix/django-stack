package scaffold

import "fmt"

func (e *Env) applyStructlog(projectPath string) error {
	if !e.Cfg.SetupStructlog {
		return nil
	}

	if err := e.installPackages(projectPath, "structlog + django-structlog", PkgStructlog, PkgDjangoStructlog); err != nil {
		return err
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}

	if !se.Contains("'" + AppStructlog + "',") {
		if err := se.AddToList(InstalledApps, AppStructlog); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", AppStructlog, InstalledApps, err)
		}
	}

	if !se.Contains(MiddlewareStructlog) {
		if err := se.AddToList(MiddlewareList, MiddlewareStructlog); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", MiddlewareStructlog, MiddlewareList, err)
		}
	}

	se.AddImport("import structlog")
	se.AppendBlock(e.structlogSettingsBlock())

	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Structured logging configured (structlog + django-structlog).")
	e.Step("Every log line carries request_id; authenticated requests also carry user_id.")
	return nil
}

func (e *Env) structlogSettingsBlock() string {
	return fmt.Sprintf(`
_LOG_RENDERER = 'plain_console' if %s else 'json_formatter'

_FOREIGN_PRE_CHAIN = [
    structlog.contextvars.merge_contextvars,
    structlog.processors.TimeStamper(fmt='iso'),
    structlog.stdlib.add_logger_name,
    structlog.stdlib.add_log_level,
    structlog.stdlib.PositionalArgumentsFormatter(),
]

LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'json_formatter': {
            '()': structlog.stdlib.ProcessorFormatter,
            'processor': structlog.processors.JSONRenderer(),
            'foreign_pre_chain': _FOREIGN_PRE_CHAIN,
        },
        'plain_console': {
            '()': structlog.stdlib.ProcessorFormatter,
            'processor': structlog.dev.ConsoleRenderer(),
            'foreign_pre_chain': _FOREIGN_PRE_CHAIN,
        },
    },
    'handlers': {
        'console': {
            'class': 'logging.StreamHandler',
            'formatter': _LOG_RENDERER,
        },
    },
    'root': {
        'handlers': ['console'],
        'level': %s,
    },
    'loggers': {
        'django_structlog': {
            'handlers': ['console'],
            'level': 'INFO',
            'propagate': False,
        },
        '%s': {
            'handlers': ['console'],
            'level': %s,
            'propagate': False,
        },
    },
}
`,
		e.env().debug(),
		e.logLevelExpression(),
		e.Cfg.ModuleName(),
		e.logLevelExpression(),
	)
}

func (e *Env) logLevelExpression() string {
	return fmt.Sprintf("'DEBUG' if %s else 'INFO'", e.env().debug())
}
