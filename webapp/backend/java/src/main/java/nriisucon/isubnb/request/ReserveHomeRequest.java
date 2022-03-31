package nriisucon.isubnb.request;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ReserveHomeRequest {
    private String homeId;
    private String userId;
    private String startDate;
    private String endDate;
    private int numberOfPeople;
}
