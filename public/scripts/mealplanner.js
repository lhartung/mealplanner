function signIn(ev) {
  ev.preventDefault();

  var data = {
    email: $("#email").val(),
    password: $("#password").val()
  };

  $.ajax({
    type: "POST",
    url: "/api/login",
    data: data
  }).then(function(response) {
    var family_id = response.families[0];
    window.location = "/family/" + family_id + "/view.html";
  }).fail(function(xhr) {
    var response = JSON.parse(xhr.responseText);
    $("#login-alert").text("There was a problem signing in: " + response.message);
    $("#login-alert").removeAttr("hidden");
  });
}

function signOut(ev) {
  ev.preventDefault();

  $.ajax({
    type: "POST",
    url: "/api/logout",
  }).then(function(response) {
    window.location = "/";
  });
}

function signUp(ev) {
  ev.preventDefault();

  var data = {
    username: $("#register-name").val(),
    email: $("#register-email").val(),
    password: $("#register-password").val()
  };

  if ($("#register-retype").val() !== data.password) {
    $("#register-alert").text("The passwords do not match.");
    $("#register-alert").removeAttr("hidden");
    return;
  }

  $.ajax({
    type: "POST",
    url: "/api/register",
    data: data
  }).then(function(response) {
    var family_id = response.families[0];
    window.location = "/family/" + family_id + "/view.html";
  }).fail(function(xhr) {
    var response = JSON.parse(xhr.responseText);
    $("#register-alert").text(response.message);
    $("#register-alert").removeAttr("hidden");
  });
}
