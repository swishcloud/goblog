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
    ajaxSubmit:function(form,option,rules){
            $(form).validate({
                rules:rules,
                submitHandler:function (form) {
                    var before=option.before
                    if(typeof before=="function")
                        before()
                    $(form).ajaxSubmit({
                        success:function (res) {
                            if(typeof res.error==="undefined"){
                                res={
                                    error:"response data format from server is invalid"
                                }
                            }
                            var success=option.success
                            if(typeof success=="function")
                                success(res)
                            else{
                                throw Error("Missing required 'success' callback function.")
                            }
                        },
                        error: function () {
                                alert("server error")
                        }
                    })
                }
            })
    }
}

$(document).ajaxStart(app.blockUi).ajaxComplete(app.unBlockUi)