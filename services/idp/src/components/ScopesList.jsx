import React from 'react';
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemText from '@material-ui/core/ListItemText';
import { withStyles } from '@material-ui/core/styles';
import PropTypes from 'prop-types';
import Checkbox from '@material-ui/core/Checkbox';

import { useTranslation } from 'react-i18next';

const styles = () => ({
  row: {
    paddingTop: 0,
    paddingBottom: 0
  }
});

const ScopesList = ({scopes, meta, classes, ...rest}) => {
  const { mapping, definitions } = meta;

  const { t } = useTranslation();

  const rows = [];
  const known = {};

  // TODO(longsleep): Sort scopes according to priority.
  for (let scope in scopes) {
    if (!scopes[scope]) {
      continue;
    }
    let id = mapping[scope];
    if (id) {
      if (known[id]) {
        continue;
      }
      known[id] = true;
    } else {
      id = scope;
    }
    let definition = definitions[id];
    let label;
    if (definition) {
      switch (definition.id) {
        case 'scope_alias_basic':
          label = t("konnect.scopeDescription.aliasBasic", "Access your basic account information");
          break;
        case 'scope_offline_access':
          label = t("konnect.scopeDescription.offlineAccess", "Keep the allowed access persistently and forever");
          break;
        default:
      }
      if (!label) {
        label = definition.description;
      }
    }
    if (!label) {
      label = t("konnect.scopeDescription.scope", "Scope: {{scope}}", { scope });
    }

    rows.push(
      <ListItem
        disableGutters
        dense
        key={id}
        className={classes.row}
      ><Checkbox
          checked
          disableRipple
          disabled
        />
        <ListItemText primary={label} />
      </ListItem>
    );
  }

  return (
    <List {...rest}>
      {rows}
    </List>
  );
};

ScopesList.propTypes = {
  classes: PropTypes.object.isRequired,

  scopes: PropTypes.object.isRequired,
  meta: PropTypes.object.isRequired
};

export default withStyles(styles)(ScopesList);
