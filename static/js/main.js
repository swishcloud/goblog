var app = {
    blockUi: function () {
        $("#mask").removeClass("d-none")
    },

    unBlockUi: function () {
        $("#mask").addClass("d-none")
    },

    showError: function (target, error) {
        $(target).append($('<div class="alert alert-danger" role="alert"></div>').text(error))
    },
    closeError: function (target) {
        $(target).find("div:first").alert("close")
    },
    ajaxSubmit:function(form,option){
            $(form).validate({
                submitHandler:function (form) {
                    var before=option.before
                    if(typeof before=="function")
                        before()
                    $(form).ajaxSubmit({
                        success:function (res) {
                            var success=option.success
                            if(typeof success=="function")
                                success(res)
                            else{
                                throw Error("Missing required 'success' callback function.")
                            }
                        }
                    })
                }
            })
    }
}

$(document).ajaxStart(app.blockUi).ajaxComplete(app.unBlockUi)