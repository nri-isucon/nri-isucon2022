package nriisucon.isubnb.request;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ReserveActivityRequest {
    private String activityId;
    private String userId;
    private String date;
    private int numberOfPeople;
}
