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
# =============================================================================
# Structured Logging
# =============================================================================
# Read straight from the environment rather than the DEBUG name: the multi-env
# split moves DEBUG out of base.py, and this block stays behind in it.
_LOG_RENDERER = 'plain_console' if %s else 'json_formatter'

# foreign_pre_chain is what carries request_id onto records emitted by loggers
# that are not structlog's own — django.request and django.server among them.
# Without it the correlation is missing from exactly the lines you reach for
# when a request goes wrong.
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
    # Logs go to stdout only. Anything that collects them — a container runtime,
    # systemd, a platform log drain — reads them from there, and no log file
    # ends up in the repository.
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

structlog.configure(
    processors=[
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.filter_by_level,
        structlog.processors.TimeStamper(fmt='iso'),
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
    ],
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)
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
