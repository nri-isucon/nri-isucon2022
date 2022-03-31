
package nriisucon.isubnb.response;

import java.util.List;
import lombok.Data;
import nriisucon.isubnb.model.Activity;

@Data
public class ActivityResponse {

    private int count;
    private List<Activity> activities;

    public ActivityResponse(List<Activity> activity) {
        this.count = activity.size();
        this.activities = activity;
    }
}
