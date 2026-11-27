import {
  RECEIVE_VALIDATE_LOGON,
  REQUEST_LOGON,
  RECEIVE_LOGON,
  RECEIVE_LOGOFF,
  REQUEST_CONSENT_ALLOW,
  REQUEST_CONSENT_CANCEL,
  RECEIVE_CONSENT,
  UPDATE_INPUT,
  RESET_MFA
} from '../actions/types';

function loginReducer(state = {
  loading: '',
  username: '',
  password: '',
  code: '',
  mfaRequired: false,
  errors: {}
}, action) {
  switch (action.type) {
    case RECEIVE_VALIDATE_LOGON:
      return Object.assign({}, state, {
        errors: action.errors,
        loading: ''
      });

    case REQUEST_CONSENT_ALLOW:
    case REQUEST_CONSENT_CANCEL:
    case REQUEST_LOGON:
      return Object.assign({}, state, {
        loading: action.type,
        errors: {}
      });

    case RECEIVE_CONSENT:
      return Object.assign({}, state, {
        errors: !action.success && action.errors ? action.errors : {},
        loading: ''
      });

    case RECEIVE_LOGON:
      return Object.assign({}, state, {
        errors: !action.success && action.errors ? action.errors : {},
        loading: '',
        // the code is single use, never keep it
        code: '',
        mfaRequired: !!action.mfaRequired
      });

    case RESET_MFA:
      return Object.assign({}, state, {
        password: '',
        code: '',
        mfaRequired: false,
        errors: {}
      });

    case RECEIVE_LOGOFF:
      return Object.assign({}, state, {
        username: '',
        password: '',
        code: '',
        mfaRequired: false
      });

    case UPDATE_INPUT:
      delete state.errors[action.name];
      return Object.assign({}, state, {
        [action.name]: action.value
      });

    default:
      return state;
  }
}

export default loginReducer;
