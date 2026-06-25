const userConfig = JSON.parse(open('./users.json'));

export function configuredUserCount() {
  return Number(__ENV.USER_COUNT || userConfig.count || '0');
}

export function userOffset() {
  return Number(__ENV.USER_OFFSET || '0');
}

export function userFromNumber(userNumber) {
  return {
    number: userNumber,
    name: `${userConfig.namePrefix} ${userNumber}`,
    email: `${userConfig.emailPrefix}-${userNumber}@${userConfig.emailDomain}`,
    password: userConfig.password,
  };
}

export function userForVU(vuID) {
  return userFromNumber(userOffset() + vuID);
}

export function validateUserCapacity(requiredUsers) {
  const availableUsers = configuredUserCount() - userOffset();
  if (requiredUsers > availableUsers) {
    throw new Error(`Need ${requiredUsers} users, but users.json only provides ${availableUsers}. Increase USER_COUNT or lower VUS.`);
  }
}
